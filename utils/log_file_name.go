package utils

import "strings"

func NewLogFileName(serverName string, addr string) string {
	return serverName + "-" + strings.ReplaceAll(addr, ":", "_")
}
