package utils

import "time"

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalText(text []byte) error {
	return d.FromString(string(text))
}

func (d *Duration) MarshalText() string {
	return d.Duration.String()
}

//FromString 从字符串解析Duration，例如：5s,7m,9ms
func (d *Duration) FromString(str string) error {
	var err error
	d.Duration, err = time.ParseDuration(str)

	return err
}
