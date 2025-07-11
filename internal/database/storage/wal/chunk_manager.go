package wal

import (
	"bufio"
	"fmt"
	"github.com/TimonKK/inmemory-db/internal/database/fs"
	"iter"
	"os"
	"path"
	"slices"

	"go.uber.org/zap"
)

type FileManager interface {
	OpenFile(string) (fs.File, error)
	WriteFile(string, []byte) error
	ReadDir(string) ([]os.DirEntry, error)
	Lines(string) iter.Seq2[string, error]
	Reset() error
}

// ChunkManager - управление всеми wal-файлами: открытие, запись, ротация
type ChunkManager struct {
	fileManager    FileManager
	dir            string
	maxSegmentSize int
	num            int
	logger         *zap.Logger
	lastChunkFile  fs.File
	lastChunk      *bufio.Writer
}

func NewChunkManager(fileManager FileManager, dir string, maxSegmentSize int, logger *zap.Logger) *ChunkManager {
	return &ChunkManager{
		fileManager:    fileManager,
		logger:         logger,
		dir:            dir,
		maxSegmentSize: maxSegmentSize,
	}
}

func (c *ChunkManager) rotate() error {
	if err := c.closeLastChunk(); err != nil {
		return err
	}

	c.num++
	latestWalFile := fmt.Sprintf(FormatWalFilename, c.num)
	file, err := c.fileManager.OpenFile(path.Join(c.dir, latestWalFile))
	if err != nil {
		return err
	}

	c.lastChunkFile = file
	c.lastChunk = bufio.NewWriter(file)

	c.logger.Info("rotate wal file", zap.String("latestWalFile", latestWalFile))

	return nil
}

func (c *ChunkManager) yieldLines(fileNames []string, yield func(string, error) bool) {
	for _, fileName := range fileNames {
		for data, err := range c.fileManager.Lines(path.Join(c.dir, fileName)) {
			if err != nil {
				yield("", err)
				return
			}
			if !yield(data, nil) {
				return
			}
		}
	}
}

func (c *ChunkManager) closeLastChunk() error {
	if err := c.Flush(); err != nil {
		return err
	}

	if c.lastChunkFile != nil {
		if err := c.lastChunkFile.Close(); err != nil {
			return err
		}
	}

	return nil
}

// All — все строки всех WAL-файлов
func (c *ChunkManager) All() iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		files, err := c.GetWalFiles()
		if err != nil {
			yield("", err)
			return
		}

		c.yieldLines(files, yield)
	}
}

// GetChunkLines — все строки одного файла
func (c *ChunkManager) ChunkLines(fileName string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		c.yieldLines([]string{fileName}, yield)
	}
}

func (c *ChunkManager) Size() int {
	return c.lastChunk.Size()
}

func (c *ChunkManager) Open() error {
	// получить список файлов вида wal.N.log
	latestWalFile := ""
	files, err := c.GetWalFiles()
	if err != nil {
		return err
	}

	for _, walFile := range files {
		lastestN := GetWalNum(latestWalFile)
		walFileN := GetWalNum(walFile)
		if walFileN > lastestN {
			c.num = walFileN
			latestWalFile = walFile
		}
	}

	if latestWalFile == "" {
		c.num = 0
		latestWalFile = DefaultWalFilename
	}

	// close old last chunk
	if err := c.closeLastChunk(); err != nil {
		return err
	}

	// open new
	file, err := c.fileManager.OpenFile(path.Join(c.dir, latestWalFile))
	if err != nil {
		return err
	}

	c.lastChunkFile = file
	c.lastChunk = bufio.NewWriter(file)

	c.logger.Info("opening wal file", zap.String("file", latestWalFile))

	return nil
}

func (c *ChunkManager) GetWalFiles() ([]string, error) {
	files, err := c.fileManager.ReadDir(c.dir)
	if err != nil {
		return nil, err
	}

	onlyWalFiles := make([]string, 0)
	for _, file := range files {
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
			onlyWalFiles = append(onlyWalFiles, file.Name())
		}
	}

	slices.SortFunc(onlyWalFiles, func(a, b string) int {
		aN := GetWalNum(a)
		bN := GetWalNum(b)
		return aN - bN
	})

	return onlyWalFiles, nil
}

func (c *ChunkManager) Save(name string, data []string) error {
	file, err := c.fileManager.OpenFile(path.Join(c.dir, name))
	if err != nil {
		return err
	}
	defer func(file fs.File) {
		if err := file.Close(); err != nil {
			c.logger.Error("failed to close file", zap.String("file", name), zap.Error(err))
		}
	}(file)

	writer := bufio.NewWriter(file)

	for _, line := range data {
		_, err = writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}

	return writer.Flush()
}

func (c *ChunkManager) Write(data []byte) error {
	if c.lastChunk.Size() >= c.maxSegmentSize {
		if err := c.rotate(); err != nil {
			return err
		}
	}

	_, err := c.lastChunk.Write(data)
	if err != nil {
		return err
	}

	c.logger.Info("ChunkManager Write", zap.Int("bites", len(data)))

	return nil
}

func (c *ChunkManager) Flush() error {
	if c.lastChunkFile == nil {
		return nil
	}

	return c.lastChunk.Flush()
}

func (c *ChunkManager) Reset() error {
	if err := c.closeLastChunk(); err != nil {
		return err
	}

	return c.fileManager.Reset()
}
