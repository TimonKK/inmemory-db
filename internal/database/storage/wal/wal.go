package wal

import (
	"context"
	"iter"
	"sync"
	"time"

	"github.com/TimonKK/inmemory-db/internal/config"
	"github.com/TimonKK/inmemory-db/internal/utils"
	"go.uber.org/zap"
)

type walRecord struct {
	data    string
	promise utils.Promise[error]
}

type WAL struct {
	config       *config.WALConfig
	logger       *zap.Logger
	chunkManager *ChunkManager

	mu      sync.RWMutex
	batch   []walRecord
	batchCh chan []walRecord
}

func NewWAL(chunkManager *ChunkManager, config *config.WALConfig, logger *zap.Logger) *WAL {
	w := WAL{
		config:       config,
		logger:       logger,
		chunkManager: chunkManager,
		batch:        make([]walRecord, 0, config.FlushingBatchSize),
		batchCh:      make(chan []walRecord, 1),
	}

	return &w
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

// TODO придумать что делать, если вызов segment.Write или Flush вернули ошибку:
// - ничего не делать
// - отправтиь дальше, не трогая promises
// - отправтиь дальше, передав ее в promises - только какой смысл?
func (w *WAL) flushBatch(batch []walRecord) error {
	if len(batch) == 0 {
		return nil
	}

	w.logger.Info("FlushData: start", zap.Int("batchSize", len(batch)), zap.Int("segmentSize", w.chunkManager.Size()))

	promises := make([]utils.Promise[error], 0, len(batch))
	for _, walRecord := range batch {
		err := w.chunkManager.Write([]byte(walRecord.data + "\n"))
		if err != nil {
			return err
		}

		promises = append(promises, walRecord.promise)
	}

	err := w.chunkManager.Flush()
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

func (w *WAL) Start(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if err := w.chunkManager.Open(); err != nil {
		return err
	}

	// TODO добавить явную обработку ошибок и выход при ее наступлении
	w.startBackgroundWorker(ctx)

	return nil
}

// All - итератор по данным wal-файлов. Читает файл по порядку, каждая строка отделена \n
func (w *WAL) All() iter.Seq2[string, error] {
	return w.chunkManager.All()
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

func (w *WAL) GetWalFiles() ([]string, error) {
	return w.chunkManager.GetWalFiles()
}
