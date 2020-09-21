// +build !windows,!plan9

package ulimit

import (
	"sync"
	"syscall"
)

var rLimit sync.Once

func SetRLimit() error {
	var err error

	rLimit.Do(func() {
		err = setRLimit()
	})

	return err
}

func setRLimit() error {
	var rLimit syscall.Rlimit

	rLimit.Cur = 999999
	rLimit.Max = 999999

	err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		return err
	}

	return err
}
