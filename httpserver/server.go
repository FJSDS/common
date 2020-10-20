package httpserver

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"

	"github.com/FJSDS/common/logger"
	"github.com/FJSDS/common/ulimit"
	"github.com/FJSDS/common/utils"
)

type Service struct {
	cfg *HttpConfig
	log *logger.Logger
	fs  *echo.Echo
}

func NewService(log *logger.Logger, cfg *HttpConfig) *Service {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.CORS())
	e.Use(middleware.GzipWithConfig(middleware.GzipConfig{Skipper: middleware.DefaultSkipper, Level: 9}))
	s := &Service{
		cfg: cfg,
		log: log,
		fs:  e,
	}
	return s
}

func (this_ *Service) GetRouter() *echo.Echo {
	return this_.fs
}

var reqBodyPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}

func GetBodyBytes() []byte {
	return reqBodyPool.Get().([]byte)
}

func PutBodyBytes(data []byte) {
	reqBodyPool.Put(data)
}

func (this_ *Service) ReadReqBody(ctx echo.Context, out []byte) (n int64, err error) {
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			err = fmt.Errorf("%v", e)
		}
	}()
	outLen := int64(len(out))
	for {
		tempOut := out[n:]
		m, e := ctx.Request().Body.Read(tempOut)
		if m < 0 {
			return n, fmt.Errorf("read error n:%d", m)
		}
		n += int64(m)
		if n >= outLen {
			return n, fmt.Errorf("read error,body to large:%d", n)
		}
		if e == io.EOF {
			return n, nil // e is EOF, so return nil explicitly
		}
		if e != nil {
			return n, e
		}
	}
}

func RealIP(ctx *fasthttp.RequestCtx) string {
	if ip := ctx.Request.Header.Peek("X-Forwarded-For"); len(ip) > 0 {
		return strings.Split(utils.BytesToString(ip), ", ")[0]
	}
	if ip := ctx.Request.Header.Peek("X-Real-IP"); len(ip) > 0 {
		return utils.BytesToString(ip)
	}

	return ctx.RemoteIP().String()
}

func (this_ *Service) Serve() error {
	err := ulimit.SetRLimit()
	if err != nil {
		return err
	}
	this_.log.Info("http start...", zap.Any("cfg", this_.cfg))
	if this_.cfg.IsTLS {
		if this_.cfg.TLSCrtFile != "" && this_.cfg.TLSKeyFile != "" {
			return this_.fs.StartTLS(this_.cfg.Addr, this_.cfg.TLSCrtFile, this_.cfg.TLSKeyFile)
		}
		return this_.fs.StartAutoTLS(this_.cfg.Addr)
	}

	return this_.fs.Start(this_.cfg.Addr)
}

func (this_ *Service) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	return this_.fs.Shutdown(ctx)
}
