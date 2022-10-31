package middleware

import (
	"time"
)

type CallStat struct {
	MaxDuration     time.Duration `json:"max_duration"`
	MinDuration     time.Duration `json:"min_duration"`
	AvgDuration     time.Duration `json:"avg_duration"`
	TotalCount      int64         `json:"total_count"`
	SuccessfulCount int64         `json:"successful_count"`
	FailureCount    int64         `json:"failure_count"`
}

//func RuntimeStat[I flex.Injector](m util.Map[string, *CallStat]) flex.Wrapper[I] {
//	return func(inner flex.Handler[I]) flex.Handler[I] {
//
//	}
//}
