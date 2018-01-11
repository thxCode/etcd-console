package backend

import (
	"time"
)

type JSONTime time.Time

const (
	timeFormat = "2006-01-02 15:04:05"
)

func (t *JSONTime) UnmarshalJSON(data []byte) (err error) {
	now, err := time.ParseInLocation(`"`+timeFormat+`"`, string(data), time.Local)
	*t = JSONTime(now)
	return
}

func (t JSONTime) MarshalJSON() ([]byte, error) {
	b := make([]byte, 0, len(timeFormat)+2)
	b = append(b, '"')
	b = time.Time(t).AppendFormat(b, timeFormat)
	b = append(b, '"')
	return b, nil
}

func (t JSONTime) Unix() int64 {
	return time.Time(t).Unix()
}

func RoundDownDuration(d, scale time.Duration) time.Duration {
	d /= scale // round down in scale
	d *= scale
	return d
}
