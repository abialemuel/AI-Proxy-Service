package time

import "time"

func GetOffsetTime() int {
	t := time.Now()
	_, offset := t.Zone()
	return offset
}
