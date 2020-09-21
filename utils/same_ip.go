package utils

import (
	"net"
	"strconv"
)

func CheckIsSameIP(addr string, selfIPS ...string) string {
	ta, err := net.ResolveTCPAddr("tcp4", addr)
	if err != nil {
		return addr
	}
	ip := ta.IP.String()
	for _, v := range selfIPS {
		if ip == v {
			return "127.0.0.1" + ":" + strconv.Itoa(ta.Port)
		}
	}
	return addr
}
