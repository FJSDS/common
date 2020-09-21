package timer

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestTimer(t *testing.T) {
	fmt.Println(Now())
	start := Now()
	time.Sleep(time.Second)
	end := Now()
	temp := end.Sub(start) / time.Millisecond
	if temp > 1001 || temp < 999 {
		require.Fail(t, "temp must in [999,1001]")
	}
	start = Now()
	time.Sleep(time.Millisecond * 2048)
	end = Now()
	temp = end.Sub(start) / time.Millisecond
	if temp > 2049 || temp < 2047 {
		require.Fail(t, "temp must in [2047,2049]")
	}
}
