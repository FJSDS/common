package logger

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestLogger(t *testing.T) {
	r := require.New(t)
	log, err := NewLogger("test", "./log", zap.DebugLevel, WithFile())
	r.NoError(err)
	defer log.Flush()
	log.Info("test", zap.Int64("int64", int64(222)))
}
