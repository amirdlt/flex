package middleware

import (
	. "github.com/amirdlt/flex/flx"
	"net"
	"sync"
	"time"
)

type (
	LimitKeyGenerator[I Injector] func(i I) string
	limiter[I Injector]           struct {
		maxCount     int
		index        int
		interval     time.Duration
		requestTimes map[string][]time.Duration
		keyGenerator LimitKeyGenerator[I]
		*sync.RWMutex
	}
)

func (l *limiter[I]) isAllowed(id string) bool {
	l.RLock()
	defer l.RUnlock()

	if l.index <= l.maxCount {
		return true
	}

	requestTimeList := l.requestTimes[id]

	var latest time.Duration
	index := l.index % (l.maxCount + 1)
	if index == 0 {
		latest = requestTimeList[l.maxCount]
	} else {
		latest = requestTimeList[index-1]
	}

	return latest-requestTimeList[index] > l.interval
}

func (l *limiter[I]) requestReceived(received time.Duration, id string) {
	l.Lock()

	requestTimeList, exist := l.requestTimes[id]
	if !exist {
		requestTimeList = make([]time.Duration, l.maxCount+1)
		l.requestTimes[id] = requestTimeList
	}

	_len := len(requestTimeList)

	if _len < l.maxCount {
		requestTimeList[_len] = received
	} else {
		requestTimeList[l.index%(l.maxCount+1)] = received
	}

	l.index++

	l.Unlock()
}

func CustomLimiter[I Injector](limitKeyGenerator LimitKeyGenerator[I], maxCount int, interval time.Duration, rateExceededHandler ...Handler[I]) Wrapper[I] {
	l := &limiter[I]{
		maxCount:     maxCount,
		interval:     interval,
		index:        0,
		requestTimes: map[string][]time.Duration{},
		keyGenerator: limitKeyGenerator,
		RWMutex:      &sync.RWMutex{},
	}

	var reh Handler[I]
	switch len(rateExceededHandler) {
	case 0:
		reh = func(i I) Result {
			return i.WrapTooManyRequestsErr("too many requests")
		}
	case 1:
		reh = rateExceededHandler[0]
	default:
		panic("rate limit exceeded handler should be one at max")
	}

	return func(h Handler[I]) Handler[I] {
		return func(i I) Result {
			id := l.keyGenerator(i)
			l.requestReceived(time.Duration(time.Now().UnixNano()), id)
			if !l.isAllowed(id) {
				return reh(i)
			}

			return h(i)
		}
	}
}

func DosLimiter[I Injector](maxCount int, interval time.Duration, rateExceededHandler Handler[I]) Wrapper[I] {
	return CustomLimiter(func(i I) string {
		ip, _, _ := net.SplitHostPort(i.RemoteAddr())
		return ip
	}, maxCount, interval, rateExceededHandler)
}
