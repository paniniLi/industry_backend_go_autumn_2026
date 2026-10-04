package main

import (
	"sync"
	"time"
)

type Clock interface{ Now() time.Time }
type Limiter struct {
	mu     sync.Mutex
	clock  Clock
	rate   float64
	burst  int
	tokens float64
	last   time.Time
}

func NewLimiter(clock Clock, ratePerSec float64, burst int) *Limiter {
	var last time.Time
	if clock != nil {
		last = clock.Now()
	}
	return &Limiter{clock: clock, rate: ratePerSec, burst: burst, tokens: float64(burst), last: last}
}

func (l *Limiter) AllowN(n int) bool {
	l.mu.Lock()
	if l.burst <= 0 || l.clock == nil {
		/* лимитер выключен, токены не пополняются и не списываются */
		l.mu.Unlock()
		return false
	}

	/* пополнение токенов */
	if l.rate > 0 {
		var now = l.clock.Now()
		l.tokens = min(l.tokens+(l.clock.Now().Sub(l.last).Seconds())*l.rate, float64(l.burst)) //защита от переполнения корзины
		l.last = now
	}

	/* списание токенов */
	var n2float = float64(n)
	if n2float <= 0 || n > l.burst || l.tokens < n2float {
		/* если недостаточно токенов, либо величина списания невалидна -> ничего не списываем */
		l.mu.Unlock()
		return false
	}
	l.tokens -= n2float
	l.mu.Unlock()
	return true
}
