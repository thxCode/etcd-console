package utils

import "time"

func RoundDownDuration(d, scale time.Duration) time.Duration {
	d /= scale // round down in scale
	d *= scale
	return d
}
