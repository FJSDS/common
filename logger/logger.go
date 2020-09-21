package logger

import (
	"path/filepath"
	"sync/atomic"
	"unsafe"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/FJSDS/common/timer"
	"github.com/FJSDS/common/utils"
)

var zero0 = int64(0)

var log *Logger

func SetDefaultLogger(l *Logger) {
	log = l
}

func GetDefaultLogger() *Logger {
	return log
}

func InitDefaultLogger(name, path string, lvl zapcore.Level, opts ...Option) (*Logger, error) {
	var err error
	log, err = NewLogger(name, path, lvl, opts...)
	return log, err
}

type Option func(o *Options)

//
func WithStderr() Option {
	return func(o *Options) {
		o.IsStderr = true
	}
}

func WithStdout() Option {
	return func(o *Options) {
		o.IsStdout = true
	}
}

func WithFile() Option {
	return func(o *Options) {
		o.IsFile = true
	}
}

func WithExpired(day int64) Option {
	return func(o *Options) {
		o.ExpiredDay = day
	}
}

type Options struct {
	IsStderr   bool
	IsStdout   bool
	IsFile     bool
	ExpiredDay int64
}

func NewLogger(name, path string, lvl zapcore.Level, opts ...Option) (*Logger, error) {
	err := utils.CheckAndCreate(path)
	if err != nil {
		return nil, err
	}

	o := &Options{}
	for _, v := range opts {
		v(o)
	}
	if !o.IsFile && !o.IsStdout && !o.IsStderr {
		o.IsStdout = true
	}
	if o.ExpiredDay <= 0 {
		o.ExpiredDay = 7
	}

	l := &Logger{
		tomorrow: unsafe.Pointer(&zero0),
		name:     name,
		path:     path,
		lvl:      lvl,
		opts:     o,
	}
	l.checkTomorrow()
	l.deleteExpiredLog(o.ExpiredDay)
	return l, nil
}

func newLogger(path string, lvl zapcore.Level, opts *Options) (*zap.Logger, error) {
	ec := zap.NewDevelopmentEncoderConfig()

	var outPath []string
	if opts.IsStdout {
		outPath = append(outPath, "stdout")
	}
	if opts.IsStderr {
		outPath = append(outPath, "stderr")
	}
	if opts.IsFile {
		outPath = append(outPath, path)
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(lvl),
		Development:      false,
		Encoding:         "console",
		EncoderConfig:    ec,
		OutputPaths:      outPath,
		ErrorOutputPaths: []string{"stderr"},
	}

	return cfg.Build(zap.AddStacktrace(zap.ErrorLevel), zap.AddCallerSkip(1))
}

type Logger struct {
	log      unsafe.Pointer
	sugar    unsafe.Pointer
	tomorrow unsafe.Pointer
	name     string
	path     string
	lvl      zapcore.Level
	opts     *Options
}

func (this_ *Logger) Flush() {
	_ = this_.getLog().Sync()
}

func (this_ *Logger) checkTomorrow() {
	t := timer.Now()
	//tomorrowStr := t.Format("2006_01_02")
	tomorrowDay := int64(t.YearDay())
	if tomorrowDay != *(*int64)(atomic.LoadPointer(&this_.tomorrow)) {
		if atomic.CompareAndSwapPointer(&this_.tomorrow, this_.tomorrow, unsafe.Pointer(&tomorrowDay)) {
			pathFile := filepath.Join(this_.path, t.Format("2006_01_02")+"_"+this_.name+".log")
			log, err := newLogger(pathFile, this_.lvl, this_.opts)
			if err != nil {
				panic(err)
				return
			}

			l := (*zap.Logger)(atomic.LoadPointer(&this_.log))
			if l != nil {
				_ = l.Sync()
			}

			atomic.StorePointer(&this_.log, unsafe.Pointer(log))
			atomic.StorePointer(&this_.sugar, unsafe.Pointer(log.Sugar()))
		}
	}
}

func (this_ *Logger) getLog() *zap.Logger {
	this_.checkTomorrow()
	return (*zap.Logger)(atomic.LoadPointer(&this_.log))
}

func (this_ *Logger) getSugar() *zap.SugaredLogger {
	this_.checkTomorrow()
	return (*zap.SugaredLogger)(atomic.LoadPointer(&this_.sugar))
}

func (this_ *Logger) Debug(msg string, fields ...zap.Field) {
	this_.getLog().Debug(msg, fields...)
}

func (this_ *Logger) Info(msg string, fields ...zap.Field) {
	this_.getLog().Info(msg, fields...)
}

func (this_ *Logger) Warn(msg string, fields ...zap.Field) {
	this_.getLog().Warn(msg, fields...)
}

func (this_ *Logger) Error(msg string, fields ...zap.Field) {
	this_.getLog().Error(msg, fields...)
}

func (this_ *Logger) DPanic(msg string, fields ...zap.Field) {
	this_.getLog().DPanic(msg, fields...)
}

func (this_ *Logger) Panic(msg string, fields ...zap.Field) {
	this_.getLog().Panic(msg, fields...)
}

func (this_ *Logger) Fatal(msg string, fields ...zap.Field) {
	this_.getLog().Fatal(msg, fields...)
}

func (this_ *Logger) DebugFormat(format string, args ...interface{}) {
	this_.getSugar().Debugf(format, args...)
}

func (this_ *Logger) InfoFormat(format string, args ...interface{}) {
	this_.getSugar().Infof(format, args...)
}

func (this_ *Logger) WarnFormat(format string, args ...interface{}) {
	this_.getSugar().Warnf(format, args...)
}

func (this_ *Logger) ErrorFormat(format string, args ...interface{}) {
	this_.getSugar().Errorf(format, args...)
}

func (this_ *Logger) DPanicFormat(format string, args ...interface{}) {
	this_.getSugar().DPanicf(format, args...)
}

func (this_ *Logger) PanicFormat(format string, args ...interface{}) {
	this_.getSugar().Panicf(format, args...)
}

func (this_ *Logger) FatalFormat(format string, args ...interface{}) {
	this_.getSugar().Fatalf(format, args...)
}

func Debug(msg string, fields ...zap.Field) {
	log.getLog().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	log.getLog().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	log.getLog().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	log.getLog().Error(msg, fields...)
}

func DPanic(msg string, fields ...zap.Field) {
	log.getLog().DPanic(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	log.getLog().Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	log.getLog().Fatal(msg, fields...)
}

func Flush() {
	log.Flush()
}

func DebugFormat(format string, args ...interface{}) {
	log.getSugar().Debugf(format, args...)
}

func InfoFormat(format string, args ...interface{}) {
	log.getSugar().Infof(format, args...)
}

func WarnFormat(format string, args ...interface{}) {
	log.getSugar().Warnf(format, args...)
}

func ErrorFormat(format string, args ...interface{}) {
	log.getSugar().Errorf(format, args...)
}

func DPanicFormat(format string, args ...interface{}) {
	log.getSugar().DPanicf(format, args...)
}

func PanicFormat(format string, args ...interface{}) {
	log.getSugar().Panicf(format, args...)
}

func FatalFormat(format string, args ...interface{}) {
	log.getSugar().Fatalf(format, args...)
}
