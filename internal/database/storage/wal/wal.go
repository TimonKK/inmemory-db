package wal

import (
	"bufio"
	"context"
	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/database/compute"
	"github.com/TimonKK/inmemory-db/internal/utils"
	"go.uber.org/zap"
	"os"
	"path"
	"slices"
	"sync"
	"time"
)

type walRecord struct {
	data    string
	promise utils.Promise[error]
}

type WAL struct {
	config *config.WALConfig
	logger *zap.Logger

	mu      sync.RWMutex
	segment *Segment

	batch   []walRecord
	batchCh chan []walRecord
}

func NewWAL(config *config.WALConfig, logger *zap.Logger) *WAL {
	w := WAL{
		config:  config,
		logger:  logger,
		segment: NewSegment(config.DataDirectory, int(config.MaxSegmentSize)),
		batch:   make([]walRecord, 0, config.FlushingBatchSize),
		batchCh: make(chan []walRecord, 1),
	}

	return &w
}

func (w *WAL) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	err := w.segment.Open()
	if err != nil {
		return err
	}

	// TODO добавить явную обработку ошибок и выход при ее наступлении
	w.startBackgroundWorker(ctx)

	return nil
}

func (w *WAL) LoadRecords() ([]compute.Query, error) {
	// получить список файлов вида wal.N.log
	walFiles := make([]string, 0)
	dir, err := os.ReadDir(w.config.DataDirectory)
	if err != nil {
		return nil, err
	}

	// пройтись по всем файлам
	for _, file := range dir {
		if file.IsDir() {
			continue
		}

		fileInfo, err := file.Info()
		if err != nil {
			return nil, err
		}

		if fileInfo.Size() == 0 {
			continue
		}

		isMatch := SegmentNameR.MatchString(file.Name())
		if isMatch {
			fileName := path.Join(w.config.DataDirectory, file.Name())
			walFiles = append(walFiles, fileName)
		}
	}

	slices.Sort(walFiles)

	records := make([]compute.Query, 0)

	for _, fileName := range walFiles {
		file, err := os.Open(fileName)
		if err != nil {
			return nil, err
		}

		// закроем только при выходе из функции, а не цикла!
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				w.logger.Error("failed to close file", zap.Error(err))
			}
		}(file)

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			query := compute.NewQueryFromString(scanner.Text())
			records = append(records, query)
		}

		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}

	return records, nil
}

func (w *WAL) startBackgroundWorker(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(w.config.FlushingBatchTimeout)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				err := w.flush()
				if err != nil {
					w.logger.Error("StartBackgroundWorker: context canceled with flush error", zap.Error(err))
				}

				return
			case <-ticker.C:
				err := w.flush()
				if err != nil {
					w.logger.Error("StartBackgroundWorker: flush error", zap.Error(err))
				}
			case batch := <-w.batchCh:
				ticker.Reset(w.config.FlushingBatchTimeout)

				err := w.flushBatch(batch)
				if err != nil {
					w.logger.Error("StartBackgroundWorker: flush batch error", zap.Error(err))
				}
			}
		}
	}()
}

// Push - отправка данных в WAL. Блокируется пока WAL не запишет данные на диск
func (w *WAL) Push(data string) error {
	p := utils.NewPromise[error]()

	w.mu.Lock()
	w.batch = append(w.batch, walRecord{data, p})
	if len(w.batch) == w.config.FlushingBatchSize {
		w.batchCh <- w.batch
		w.batch = nil
	}
	w.mu.Unlock()

	// блокируемся
	return p.Get()
}

// TODO придумать что делать, если вызов segment.Write или Flush вернули ошибку:
// - ничего не делать
// - отправтиь дальше, не трогая promises
// - отправтиь дальше, передав ее в promises - только какой смысл?
func (w *WAL) flushBatch(batch []walRecord) error {
	if len(batch) == 0 {
		return nil
	}

	w.logger.Info("FlushData: start", zap.Int("batchSize", len(batch)), zap.Int("segmentSize", w.segment.Size()))

	promises := make([]utils.Promise[error], 0, len(batch))
	for _, walRecord := range batch {
		err := w.segment.Write(walRecord.data + "\n")
		if err != nil {
			return err
		}

		promises = append(promises, walRecord.promise)
	}

	err := w.segment.Flush()
	if err != nil {
		return err
	}

	for i := 0; i < len(promises); i++ {
		promises[i].Set(nil)
	}

	w.logger.Info("FlushData: end", zap.Int("requestCount", len(promises)))

	return nil
}

func (w *WAL) flush() error {
	w.mu.Lock()
	batch := w.batch
	w.batch = nil
	w.mu.Unlock()

	return w.flushBatch(batch)
}
