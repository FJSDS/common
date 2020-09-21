package utils

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDuration_UnmarshalText(t *testing.T) {
	r := require.New(t)
	d := &Duration{}
	err := d.UnmarshalText([]byte("5s"))
	r.NoError(err)
	r.Equal(time.Second*5, d.Duration)

	err = d.UnmarshalText([]byte("5m"))
	r.NoError(err)
	r.Equal(time.Minute*5, d.Duration)

	err = d.UnmarshalText([]byte("5ms"))
	r.NoError(err)
	r.Equal(time.Millisecond*5, d.Duration)

	r.Equal("5ms", d.MarshalText())
	fmt.Println(1 << 32)
}
