package timehelper

import "time"

const (
	//默认时间格式
	Default = "2006-01-02 15:04:05"
	//只精确到小时
	Hour = "2006-01-02 15"
	//只精确到天
	Day = "2006-01-02"
	//短日期格式
	ShortDay = "20060102"
	//短时间格式
	ShortDateTime = "2006-01-02 15:04"
)

func GetZeroTime(d time.Time) time.Time {
	return time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
}

func GetNextZeroTime() time.Time {
	return GetZeroTime(time.Now().Add(24 * time.Hour))
}
