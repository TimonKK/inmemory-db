package wal

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
)

type Segment struct {
	dir            string
	maxSegmentSize int
	num            int
	buf            *bufio.Writer
}

// TODO добавить компресию файла
func NewSegment(dir string, maxSegmentSize int) *Segment {
	s := Segment{
		dir:            dir,
		maxSegmentSize: maxSegmentSize,
	}

	return &s
}

func (s *Segment) Size() int {
	return s.buf.Size()
}

func (s *Segment) Open() error {
	// получить список файлов вида wal.N.log
	latestWalFile := ""
	dir, err := os.ReadDir(s.dir)
	if err != nil {
		return err
	}

	// найти самый последний файл
	for _, file := range dir {
		if file.IsDir() {
			continue
		}

		matches := SegmentNameR.FindStringSubmatch(file.Name())
		if len(matches) < 2 {
			continue
		}

		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		if file.Name() > latestWalFile {
			s.num = num
			latestWalFile = file.Name()
		}
	}

	if latestWalFile == "" {
		s.num = 0
		latestWalFile = DefaultWalFilename
	}

	// открыть его на дозапись
	file, err := os.OpenFile(path.Join(s.dir, latestWalFile), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	s.buf = bufio.NewWriter(file)

	return nil
}

func (s *Segment) Rotate() error {
	err := s.Flush()
	if err != nil {
		return err
	}

	s.num++
	latestWalFile := fmt.Sprintf(FormatWalFilename, s.num)

	file, err := os.OpenFile(path.Join(s.dir, latestWalFile), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}

	s.buf = bufio.NewWriter(file)

	return nil
}

func (s *Segment) Write(data string) error {
	if s.buf.Size() >= s.maxSegmentSize {
		err := s.Rotate()
		if err != nil {
			return err
		}
	}

	_, err := s.buf.WriteString(data)
	if err != nil {
		return err
	}

	return nil
}

func (s *Segment) Flush() error {
	return s.buf.Flush()
}
