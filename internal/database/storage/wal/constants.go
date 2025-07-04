package wal

import (
	"fmt"
	"regexp"
)

var (
	FormatWalFilename  = "wal.%d.log"
	DefaultWalFilename = fmt.Sprintf(FormatWalFilename, 0)
	SegmentNameR       = regexp.MustCompile(`wal.(\d).log`)
)
