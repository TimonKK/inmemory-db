package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var (
	ErrEmptySizeStr   = errors.New("empty size string")
	ErrNonNumericSize = errors.New("no numeric value found")
)

// ParseSizeString преобразует строку вида "4KB" в байты
func ParseSizeString(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))
	if sizeStr == "" {
		return 0, ErrEmptySizeStr
	}

	// Отделяем число от единицы измерения
	i := 0
	for ; i < len(sizeStr); i++ {
		if sizeStr[i] < '0' || sizeStr[i] > '9' {
			break
		}
	}

	if i == 0 {
		return 0, ErrNonNumericSize
	}

	num, err := strconv.ParseInt(sizeStr[:i], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number: %w", err)
	}

	unit := strings.TrimSpace(sizeStr[i:])

	switch unit {
	case "", "B":
		return num, nil
	case "KB":
		return num * 1024, nil
	case "MB":
		return num * 1024 * 1024, nil
	case "GB":
		return num * 1024 * 1024 * 1024, nil
	case "TB":
		return num * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}
}
