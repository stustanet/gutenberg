package main

import (
	"sync"
	"sync/atomic"
	"time"
)

const (
	rateThreshold = 4
	waitSeconds   = 60
	gcRate        = 5
)

// TODO: non-linear penalty, e.g. double

type RateLimiter struct {
	lock sync.RWMutex
	rate map[string]*uint64
	gc   uint
	stop chan struct{}
}

func NewRateLimiter() (r *RateLimiter) {
	r = new(RateLimiter)
	r.rate = make(map[string]*uint64)
	r.stop = make(chan struct{})
	go r.tick()
	return
}

func (r *RateLimiter) Request(key string) bool {
	r.lock.RLock()
	if cp, ok := r.rate[key]; !ok {
		// we need to acquire a write lock instead
		// the entry might have been created in the meantime
		r.lock.RUnlock()
		r.lock.Lock()
		if cp, ok = r.rate[key]; !ok {
			cp = new(uint64)
			r.rate[key] = cp
		} else if *cp >= rateThreshold {
			r.lock.Unlock()
			return false
		}

		*cp++
		r.lock.Unlock()
		return true
	} else {
		for {
			count := atomic.LoadUint64(cp)
			if count >= rateThreshold {
				r.lock.RUnlock()
				return false
			}

			if atomic.CompareAndSwapUint64(cp, count, count+1) {
				r.lock.RUnlock()
				return true
			}
		}
	}
}

func (r *RateLimiter) Stop() {
	r.lock.Lock()
	if r.stop != nil {
		close(r.stop)
	}
	r.lock.Unlock()
}

func (r *RateLimiter) tick() {
	ticker := time.NewTicker(waitSeconds * time.Second)
	for {
		select {
		case <-ticker.C:
			r.gc--
			if r.gc == 0 {
				r.lock.Lock()
				for key, cp := range r.rate {
					if *cp > 1 {
						*cp--
						continue
					}
					delete(r.rate, key)
				}
				r.lock.Unlock()
				r.gc = gcRate
			} else {
				r.lock.RLock()
				for _, cp := range r.rate {
					if atomic.LoadUint64(cp) > 0 {
						// we don't need CAS since the value can only get higher
						atomic.AddUint64(cp, ^uint64(0)) // decrement
					}
				}
				r.lock.RUnlock()
			}
		case <-r.stop:
			return
		}
	}
}
