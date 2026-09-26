package member

import (
	"fmt"
	"strconv"
)

func atoi4(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func sprintf(format string, args ...any) string {
	return fmt.Sprintf(format, args...)
}
