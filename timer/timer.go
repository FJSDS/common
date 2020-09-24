package timer

import (
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func init() {
	runTimeTicker()
	now = &timer{}
	now.run()
}

var now *timer

type timer struct {
	t atomic.Value
}

func runTimeTicker() {
	go func() {
		for {
			t := <-tempTicker
			GlobalTicker.Range(func(_, value interface{}) bool {
				v := value.(chan time.Time)
				select {
				case v <- t:
				default:
				}
				return true
			})
		}
	}()
}

var GlobalTicker sync.Map
var tempTicker = make(chan time.Time, 1)
var timerCount int64

func GetTickChan() (<-chan time.Time, int64) {
	c := make(chan time.Time, 2)
	index := atomic.AddInt64(&timerCount, 1)
	GlobalTicker.Store(index, c)
	return c, index
}

func PutTickChan(index int64) {
	v, ok := GlobalTicker.LoadAndDelete(index)
	if ok && v != nil {
		close(v.(chan time.Time))
	}
}

func (this_ *timer) run() {
	n := time.Now()
	this_.t.Store(&n)
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case t := <-ticker.C:
				this_.t.Store(&t)
				select {
				case tempTicker <- t:
				default:
				}
			}
		}
	}()
}

const bucketCount = 100

func Now() time.Time {
	return *now.t.Load().(*time.Time)
}

func AfterTimeMillisecond(d time.Duration) int64 {
	return Now().Add(d).UnixNano() / int64(time.Millisecond)
}

func NowUnixSecond() int64 {
	return Now().Unix()
}

func NowUnixNanoSecond() int64 {
	return Now().UnixNano()
}

func NowUnixMillisecond() int64 {
	return NowUnixNanoSecond() / int64(time.Millisecond)
}

func NowUnixMicrosecond() int64 {
	return NowUnixNanoSecond() / int64(time.Microsecond)
}

func ToString(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func NowToString() string {
	return ToString(Now())
}

func StringToDate(str string) (time.Time, error) {
	str = strings.Split(str, " ")[0]
	return time.Parse("2006-01-02", str)
}

func StringToDateStr(str string) (string, error) {
	t, err := StringToDate(str)
	if err != nil {
		return "", err
	}
	return ToYMDString(t), nil
}

func StringToDateStrAddDay(str string, day int64) (string, error) {
	t, err := StringToDate(str)
	if err != nil {
		return "", err
	}
	t = t.Add(time.Hour * 24 * time.Duration(day))
	return ToYMDString(t), nil
}

func StringToTime(str string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", str, time.Local)
}

func StringToTimeWithLocation(str string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", str, loc)
}

func StringToTimeStr(str string) (string, error) {
	t, e := time.ParseInLocation("2006-01-02 15:04:05", str, time.Local)
	if e != nil {
		return "", e
	}
	return ToString(t), nil
}

func NowToUnixNanoString() string {
	return strconv.Itoa(int(Now().UnixNano()))
}

func ToYMDHmsString(t time.Time) string {
	return t.Format("2006-01-02 00:00:00")
}

func ToYMDString(t time.Time) string {
	return t.Format("2006-01-02")
}

func NowAddDaysYMDHms(days int) string {
	return ToYMDHmsString(Now().Add(time.Hour * 24 * time.Duration(days)))
}

func NowAddDaysYMD(days int) string {
	return ToYMDString(Now().Add(time.Hour * 24 * time.Duration(days)))
}

// 转换当前时间为 2019-11-12 01:01:01 ---> 20191112 格式
func NowToDayString() string {
	return ToDayString(Now())
}

// 转换t为 2019-11-12 01:01:01 ---> 20191112 格式
func ToDayString(t time.Time) string {
	return t.Format("20060102")
}

func TodayStartAndEndTime() (string, string) {
	y, m, d := Now().Date()
	s := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	e := time.Date(y, m, d, 23, 59, 59, 9999, time.Local)
	return strconv.Itoa(int(s.Unix())) + "000", strconv.Itoa(int(e.Unix())) + "000"
}

// 转换当前时间为 2019-11-12 01:01:01 ---> 2019111201 格式
func NowToHourString() string {
	return ToHourString(Now())
}

// 转换t为 2019-11-12 01:01:01 ---> 2019111201 格式
func ToHourString(t time.Time) string {
	return t.Format("2006010215")
}

// 转换当前时间为 2019-11 ---> 201911 格式
func NowToMouthString() string {
	return ToMouthString(Now())
}

// 转换t为 2019-11 ---> 201911 格式
func ToMouthString(t time.Time) string {
	return t.Format("200601")
}

func AddMouth(t time.Time, value int) time.Time {
	return t.AddDate(0, value, 0)
}

//func ()  {
//	loc, _:= time.FixedZone("CST")
//	time.ParseInLocation("2006-01-02 15:04:05", "2017-05-11 14:06:06", loc)
//}

func UTCSub4ToUTCAdd8Str(str string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", str, UtcSub4Loc)
	if err != nil {
		return "", err
	}
	return ToString(t.In(time.Local)), nil
}

func UTCAdd8ToUTCSub4Str(str string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", str, time.Local)
	if err != nil {
		return "", err
	}
	return ToString(t.In(UtcSub4Loc)), nil
}

func UTCSub4ToUTCAdd8Time(utcSub4 time.Time) time.Time {
	return utcSub4.In(time.Local)
}

var UtcSub4Loc = time.FixedZone("EDT", -4*60*60)

func UTCAdd8ToUTCSub4Time(utcAdd8 time.Time) time.Time {
	return utcAdd8.In(UtcSub4Loc)
}

func UTCAdd8StrToUTCSub4Time(str string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", str, time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(UtcSub4Loc), nil
}

func UTCSub4StrToUTCAdd8Time(str string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", str, UtcSub4Loc)
	if err != nil {
		return time.Time{}, err
	}
	return t.In(time.Local), nil
}

func LastMouthStartAndEndTime(t time.Time) (string, string) {
	y, m, _ := t.Date()
	s := time.Date(y, m-1, 1, 0, 0, 0, 0, time.Local)
	e := time.Date(y, m, 0, 23, 59, 59, 9999, time.Local)
	return ToString(s), ToString(e)
}

func NowToDayInt64() int64 {
	return TimeToDayInt64(Now())
}

func TimeToDayInt64(t time.Time) int64 {
	return t.Add(time.Hour*8).Unix() / int64(3600*24)
}

func DateStrToMillsStr(date string) (string, string, error) {
	t, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return "", "", err
	}
	start := t.Unix()
	end := t.Add(time.Hour * 24).Unix()

	return strconv.Itoa(int(start * 1000)), strconv.Itoa(int((end - 1) * 1000)), nil
}

func DateStrToSecond(date string) (int64, int64, error) {
	t, err := time.ParseInLocation("2006-01-02", date, time.Local)
	if err != nil {
		return 0, 0, err
	}
	start := t.Unix()
	end := t.Add(time.Hour * 24).Unix()

	return start, end, nil
}
