package limiter

import (
	"math"
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
	"golang.org/x/time/rate"
)

type Type int

const (
	TypeRedis Type = iota << 1
	TypeInMemory
	TypeDisabled
)

// Limiter defines a rate limiter interface
type Limiter interface {
	// Allow returns true if an event may happen for the given ID
	Allow(id string) bool

	// AllowN returns true if the given number of events may happen for the
	// given ID
	AllowN(id string, n int) bool

	// AllowDynamic returns true if an event may happen for the given ID taking
	// into consideration the given rate and burst limits
	AllowDynamic(id string, rate float64, burst int) bool

	// AllowNDynamic returns true if the given number of events may happen for
	// the given ID taking into consideration the given rate and burst limits
	AllowNDynamic(id string, n int, rate float64, burst int) bool
}

// Config defines a struct passed to New to configure a Limiter
type Config struct {
	// Type defines the type of the Limiter
	Type Type
	// Address defines the Redis server address
	Address string
	// RateLimit defines the rate limit in queries per Interval
	RateLimit float64
	// BurstLimit defines the burst limit or bucket size of the Limiter
	BurstLimit int
	// Interval defines the token refresh rate of RateLimit tokens per Interval
	Interval time.Duration
	// FailOpen determines if Allow should return true on Redis server errors
	FailOpen bool
}

// redisLimiter uses redis for its storage
type redisLimiter struct {
	rate     float64
	burst    int
	interval time.Duration
	failOpen bool

	pool *redis.Pool
}

// inMemoryLimiter uses memory for its storage, useful for local development
type inMemoryLimiter struct {
	rate     float64
	burst    int
	interval time.Duration

	limiters map[string]*rate.Limiter
	mux      *sync.RWMutex
}

// disabledLimiter does not require storage, useful for unit tests
type disabledLimiter struct{}

// New creates a new redis limiter and returns an error if
// redis is not available at the configured redis address
func New(config Config) Limiter {
	// default to rate limiting on a per second interval
	if config.Interval == 0 {
		config.Interval = time.Second
	}

	switch config.Type {
	case TypeRedis:
		return &redisLimiter{
			rate:     config.RateLimit,
			burst:    config.BurstLimit,
			interval: config.Interval,
			failOpen: config.FailOpen,
			pool: &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return redis.Dial("tcp", config.Address)
				},
				TestOnBorrow: func(c redis.Conn, t time.Time) error {
					if time.Since(t) < time.Minute {
						return nil
					}
					_, err := c.Do("PING")
					return err
				},
			},
		}
	case TypeInMemory:
		return &inMemoryLimiter{
			rate:     config.RateLimit,
			burst:    int(config.BurstLimit),
			interval: config.Interval,
			limiters: make(map[string]*rate.Limiter),
			mux:      &sync.RWMutex{},
		}
	case TypeDisabled:
		return &disabledLimiter{}
	}
	return nil
}

// Allow returns true if the given key has not breached the global rate limit,
// false otherwise. Tokens are added to the bucket based on the global burst
// limit.
func (l *redisLimiter) Allow(key string) bool {
	return l.allowN(key, 1, l.rate, l.burst)
}

func (l *redisLimiter) AllowN(key string, n int) bool {
	return l.allowN(key, n, l.rate, l.burst)
}

// AllowDynamic returns true if the given key has not breached the given rate
// limit, false otherwise. Tokens are added to the bucket based on the given
// burst limit.
func (l *redisLimiter) AllowDynamic(key string, rate float64, burst int) bool {
	return l.allowN(key, 1, rate, burst)
}

func (l *redisLimiter) AllowNDynamic(key string, n int, rate float64, burst int) bool {
	return l.allowN(key, n, rate, burst)
}

// allow returns true if the given key has not breached its rate limit, false
// otherwise. In redis, the key is a list of two elements: the first is an int
// which represents the token bucket/count, the second is a unix timestamp
// which represents the last time tokens were added to the bucket.
func (l *redisLimiter) allowN(key string, n int, rate float64, burst int) bool {
	c := l.pool.Get()
	defer c.Close()

	// get list of token bucket and last token bucket update
	resp, err := redis.Values(c.Do("LRANGE", key, 0, 1))
	if err != nil {
		// fail open on redis error
		return l.failOpen
	}

	// if key doesn't exist, add it and return true
	if len(resp) == 0 {
		// truncate to rate limit on configured interval
		now := time.Now().Truncate(l.interval).Unix()
		_, err := redis.Int(c.Do("LPUSH", key, float64(burst-1), now))
		if err != nil {
			// fail open on redis error
			return l.failOpen
		}
		return true
	}

	var tokens float64
	var last int64
	if _, err := redis.Scan(resp, &tokens, &last); err != nil {
		// fail open on redis error
		return l.failOpen
	}

	// calculate how many tokens to add to the bucket
	// token allotment is the number of intervals since the last update time
	// multiplied by the rate limit
	since := time.Since(time.Unix(last, 0).Truncate(l.interval))
	allotment := float64(since/l.interval) * rate

	// calculate how many tokens we have after allotment
	// cannot have more than max bucket size tokens (burst)
	tokens = math.Min(tokens+allotment, float64(burst))

	// if we don't have tokens, return false
	if tokens < float64(n) {
		return false
	}

	// use tokens
	tokens -= float64(n)

	// truncate to rate limit on configured interval
	now := time.Now().Truncate(l.interval).Unix()

	// update the bucket and last update time
	c.Send("MULTI")
	c.Send("LSET", key, 0, tokens)
	c.Send("LSET", key, 1, now)
	_, err = c.Do("EXEC")
	if err != nil {
		// fail open on redis error
		return l.failOpen
	}

	return true
}

func (l *inMemoryLimiter) Allow(key string) bool {
	return l.allowN(key, 1, l.rate, l.burst)
}

func (l *inMemoryLimiter) AllowN(key string, n int) bool {
	return l.allowN(key, n, l.rate, l.burst)
}

func (l *inMemoryLimiter) AllowDynamic(key string, rate float64, burst int) bool {
	return l.allowN(key, 1, rate, burst)
}

func (l *inMemoryLimiter) AllowNDynamic(key string, n int, rate float64, burst int) bool {
	return l.allowN(key, n, rate, burst)
}

func (l *inMemoryLimiter) allowN(key string, n int, ratelimit float64, burst int) bool {
	l.mux.RLock()
	limiter, ok := l.limiters[key]
	l.mux.RUnlock()

	if !ok {
		l.mux.Lock()
		limiter, ok = l.limiters[key]
		if !ok {
			limiter = rate.NewLimiter(rate.Limit(ratelimit), burst)
			l.limiters[key] = limiter
		}
		l.mux.Unlock()
	}

	// truncate to rate limit on configured interval
	now := time.Now().Truncate(l.interval)

	if limiter.Burst() != burst {
		limiter.SetBurstAt(now, burst)
	}

	if limiter.Limit() != rate.Limit(ratelimit) {
		limiter.SetLimitAt(now, rate.Limit(ratelimit))
	}

	return limiter.AllowN(now, n)
}

func (l *disabledLimiter) Allow(key string) bool {
	return true
}

func (l *disabledLimiter) AllowN(key string, n int) bool {
	return true
}

func (l *disabledLimiter) AllowDynamic(key string, rate float64, burst int) bool {
	return true
}

func (l *disabledLimiter) AllowNDynamic(key string, n int, rate float64, burst int) bool {
	return true
}
