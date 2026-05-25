package application

import (
	"sync"
	"time"
)

type RateLimitService struct {
	mu      sync.Mutex
	buckets map[string]rateBucket
}

type rateBucket struct {
	Count   int
	ResetAt time.Time
}

func NewRateLimitService() *RateLimitService {
	return &RateLimitService{buckets: map[string]rateBucket{}}
}

func (r *RateLimitService) Allow(key string, max int, window time.Duration) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now().UTC()
	bucket := r.buckets[key]
	if bucket.ResetAt.IsZero() || now.After(bucket.ResetAt) {
		bucket = rateBucket{ResetAt: now.Add(window)}
	}
	bucket.Count++
	r.buckets[key] = bucket
	return bucket.Count <= max
}

func (r *RateLimitService) Reset(key string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.buckets, key)
}
