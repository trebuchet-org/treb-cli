package forge

import (
	"strconv"
	"strings"
)

// parseHexUint64 parses a hex string to uint64
func parseHexUint64(hexStr string) (uint64, bool) {
	if hexStr == "" {
		return 0, false
	}
	hexStr = strings.TrimPrefix(hexStr, "0x")
	val, err := strconv.ParseUint(hexStr, 16, 64)
	if err != nil {
		return 0, false
	}
	return val, true
}
