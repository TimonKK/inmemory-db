package fs

import (
	"bufio"

	"io/fs"
	"iter"
	"os"

	"go.uber.org/zap"
)

type FileStorage struct {
	dir    string
	logger *zap.Logger
}

func NewFileStorage(dir string, logger *zap.Logger) *FileStorage {
	return &FileStorage{
		dir:    dir,
		logger: logger,
	}
}

func (c *FileStorage) OpenFile(filePath string) (File, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (c *FileStorage) WriteFile(filePath string, data []byte) error {
	return os.WriteFile(filePath, data, 0666)
}

func (c *FileStorage) ReadDir(dir string) ([]fs.DirEntry, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	return files, nil
}

func (c *FileStorage) Lines(filePath string) iter.Seq2[string, error] {
	return func(yield func(string, error) bool) {
		file, err := os.Open(filePath)
		if err != nil {
			yield("", err)
			return
		}

		// закроем только при выходе из функции, а не цикла!
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {
				c.logger.Error("failed to close file %s", zap.Error(err), zap.String("file", file.Name()))
			}
		}(file)

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			if !yield(scanner.Text(), nil) {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			yield("", err)
			return
		}
	}
}

func (c *FileStorage) Reset() error {
	err := os.RemoveAll(c.dir)
	if err != nil {
		return err
	}

	return os.Mkdir(c.dir, 0755)
}
