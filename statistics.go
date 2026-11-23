package main

import "time"

// StatBucket is one point of a statistics graph: the local start of the hour or day and the
// value collected in it.
type StatBucket struct {
	Start int64 `json:"start"`
	Value int64 `json:"value"`
}

type StatPeriod struct {
	Total   int64        `json:"total"`
	Buckets []StatBucket `json:"buckets"`
}

// StatPeriodSet is one metric bucketed for the three ranges the statistics page shows.
type StatPeriodSet struct {
	Today StatPeriod `json:"today"`
	Week  StatPeriod `json:"week"`
	Month StatPeriod `json:"month"`
}

// Statistics is what the statistics page reads: blinks as a count, screen time in millis.
type Statistics struct {
	Blinks     StatPeriodSet `json:"blinks"`
	ScreenTime StatPeriodSet `json:"screenTime"`
}

// TodayStatistics is what the camera widget shows: one total per metric, no buckets.
type TodayStatistics struct {
	Blinks     int64 `json:"blinks"`
	ScreenTime int64 `json:"screenTime"`
}

func startOfDay(t time.Time) time.Time {
	t = t.Local()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local)
}

// A week starts on Monday: fixed week length, no locale lookup involved.
func startOfWeek(t time.Time) time.Time {
	day := startOfDay(t)
	return day.AddDate(0, 0, -((int(day.Weekday()) + 6) % 7))
}

func startOfMonth(t time.Time) time.Time {
	t = t.Local()
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
}

// buildStatistics turns hourly rows (local hour start -> value) into graph buckets. Hours
// without data are zero buckets, so every graph keeps a stable length.
func buildStatistics(now time.Time, hours map[int64]int64) StatPeriodSet {
	day := startOfDay(now)
	month := startOfMonth(now)
	daysInMonth := time.Date(month.Year(), month.Month()+1, 0, 0, 0, 0, 0, time.Local).Day()

	return StatPeriodSet{
		Today: hourBuckets(hours, day),
		Week:  dayBuckets(hours, startOfWeek(now), 7),
		Month: dayBuckets(hours, month, daysInMonth),
	}
}

// Real hour starts, not hour numbers: a DST day has 23 or 25 hours and no duplicate start.
func hourBuckets(hours map[int64]int64, day time.Time) StatPeriod {
	period := StatPeriod{Buckets: make([]StatBucket, 0, 24)}
	end := day.AddDate(0, 0, 1)

	for hour := day; hour.Before(end); hour = hour.Add(time.Hour) {
		value := sumHours(hours, hour.Unix(), hour.Add(time.Hour).Unix())

		period.Buckets = append(period.Buckets, StatBucket{Start: hour.Unix(), Value: value})
		period.Total += value
	}

	return period
}

func dayBuckets(hours map[int64]int64, start time.Time, count int) StatPeriod {
	period := StatPeriod{Buckets: make([]StatBucket, 0, count)}

	for day := range count {
		from := start.AddDate(0, 0, day)
		value := sumHours(hours, from.Unix(), from.AddDate(0, 0, 1).Unix())

		period.Buckets = append(period.Buckets, StatBucket{Start: from.Unix(), Value: value})
		period.Total += value
	}

	return period
}

func sumHours(hours map[int64]int64, from int64, to int64) int64 {
	var total int64

	for hour, value := range hours {
		if hour >= from && hour < to {
			total += value
		}
	}

	return total
}

// mergeHours adds the not yet flushed buffer into the stored rows.
func mergeHours(rows map[int64]int64, buffer map[int64]int64, from int64, to int64) {
	for hour, value := range buffer {
		if hour >= from && hour < to {
			rows[hour] += value
		}
	}
}
