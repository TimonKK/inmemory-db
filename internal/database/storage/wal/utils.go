package wal

import "strconv"

func GetWalNum(str string) int {
	matches := SegmentNameR.FindStringSubmatch(str)
	if len(matches) < 2 {
		return -1
	}

	num, err := strconv.Atoi(matches[1])
	if err != nil {
		return -1
	}

	return num
}
