package models

import "time"

type StreamMetadata struct {
	WaitingTime         time.Duration
	SkipTargetDuration  bool
	TotalDurationStream time.Duration
	StartDurationStream time.Duration
	Username, Platform  string
	SplitSegments       bool
	TimeSegment         int
}
