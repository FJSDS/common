package logger

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/FJSDS/common/timer"
	"github.com/FJSDS/common/utils"
)

var reg = regexp.MustCompile("\\d{4}_\\d{2}_\\d{2}")

func (this_ *Logger) deleteExpiredLog(day int64) {
	utils.SafeGO(func(i interface{}) {
		this_.Error("deleteExpiredLog panic", zap.Any("panic info", i))
		this_.deleteExpiredLog(day)
	}, func() {
		t := time.NewTicker(time.Hour * 1)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				this_.deleteBeforeNDay(day)
			}
		}
	})
}

func (this_ *Logger) deleteBeforeNDay(days int64) {
	files, err := utils.WalkFiles(this_.path, ".log")
	if err != nil {
		this_.Error("delete log error", zap.Error(err))
		return
	}
	for _, v := range files {
		if strings.Contains(v, this_.name) {
			dateStr := GetDate(v)
			if dateStr != "" {
				t, err := time.Parse("2006_01_02", dateStr)
				if err != nil {
					this_.Error("delete log error", zap.Error(err))
					continue
				}
				if timer.Now().Sub(t) > time.Hour*24*time.Duration(days) {
					err = os.Remove(v)
					if err != nil {
						this_.Error("delete log error", zap.Error(err))
					}
				}
			}
		}
	}
}

func GetDate(fileName string) string {
	_, file := filepath.Split(fileName)
	dates := reg.FindAllString(file, -1)
	if len(dates) > 0 {
		return dates[0]
	}
	return ""
}
