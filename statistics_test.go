package main

import (
	"testing"
	"time"
)

func TestBuildStatisticsBuckets(t *testing.T) {
	// Wednesday 18 November 2026, so the week starts on Monday the 16th
	now := time.Date(2026, time.November, 18, 13, 45, 0, 0, time.Local)
	at := func(month time.Month, day int, hour int) int64 {
		return hourStart(time.Date(2026, month, day, hour, 10, 0, 0, time.Local))
	}

	rows := map[int64]int64{
		at(time.November, 18, 9):  5,  // today
		at(time.November, 18, 13): 2,  // today, current hour
		at(time.November, 16, 8):  7,  // this week, Monday
		at(time.November, 2, 8):   11, // this month, last week
		at(time.October, 31, 23):  13, // last month, excluded
	}

	stats := buildStatistics(now, rows)

	if got := len(stats.Today.Buckets); got != 24 {
		t.Fatalf("expected 24 hourly buckets, got %d", got)
	}

	if got := len(stats.Week.Buckets); got != 7 {
		t.Fatalf("expected 7 daily buckets, got %d", got)
	}

	if got := len(stats.Month.Buckets); got != 30 {
		t.Fatalf("expected one bucket per day of November, got %d", got)
	}

	if got := time.Unix(stats.Week.Buckets[0].Start, 0).Weekday(); got != time.Monday {
		t.Fatalf("expected the week to start on Monday, got %s", got)
	}

	if stats.Today.Total != 7 {
		t.Fatalf("expected 7 blinks today, got %d", stats.Today.Total)
	}

	if stats.Week.Total != 14 {
		t.Fatalf("expected 14 blinks this week, got %d", stats.Week.Total)
	}

	if stats.Month.Total != 25 {
		t.Fatalf("expected 25 blinks this month, got %d", stats.Month.Total)
	}

	// the row lands on the bucket of its own hour, that bucket starts at the start of the hour
	if got := stats.Today.Buckets[9]; got.Value != 5 || got.Start != at(time.November, 18, 9) {
		t.Fatalf("expected 5 blinks in the 9:00 bucket, got %+v", got)
	}

	if got := stats.Today.Buckets[10].Value; got != 0 {
		t.Fatalf("expected an empty hour to be a zero bucket, got %d", got)
	}

	if got := stats.Month.Buckets[1].Value; got != 11 {
		t.Fatalf("expected 11 blinks on the 2nd, got %d", got)
	}

	if got := stats.Month.Buckets[15].Value; got != 7 {
		t.Fatalf("expected 7 blinks on the 16th, got %d", got)
	}
}

func TestBuildStatisticsEmpty(t *testing.T) {
	stats := buildStatistics(time.Now(), nil)

	if stats.Today.Total != 0 || stats.Week.Total != 0 || stats.Month.Total != 0 {
		t.Fatalf("expected empty buckets for no rows, got %+v", stats)
	}
}

func TestHourBucketsFollowTheDaylightSavingDay(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Berlin")

	if err != nil {
		t.Skip("no timezone data for Europe/Berlin")
	}

	// 29 March 2026 springs forward: 02:00 does not exist, the day has 23 hours
	day := time.Date(2026, time.March, 29, 0, 0, 0, 0, loc)
	period := hourBuckets(nil, day)

	if got := len(period.Buckets); got != 23 {
		t.Fatalf("expected 23 hourly buckets on the spring forward day, got %d", got)
	}

	seen := make(map[int64]bool)

	for _, bucket := range period.Buckets {
		if seen[bucket.Start] {
			t.Fatalf("duplicate bucket start %s", time.Unix(bucket.Start, 0).In(loc))
		}

		seen[bucket.Start] = true
	}
}
