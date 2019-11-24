package main

import (
	"fmt"
	"time"

	"github.com/blakearoberts/redis-token-bucket-rate-limiter/limiter"
)

const (
	key1 = "key1"
	key2 = "key2"
)

func main() {
	// smaller intervals may result in the final allows to return true because
	// time.Sleep will block for longer than the requested duration
	interval := 1000 * time.Millisecond

	l := limiter.New(limiter.Config{
		Type:    limiter.TypeRedis,
		Address: ":6379",

		// in this example, we allow one request per interval; however, two
		// requests can be ran in the same interval if a request was not issues
		// during the previous interval
		RateLimit:  1.0,
		BurstLimit: 2,
		Interval:   interval,

		// in this example, we do not want to FailOpen so that we can detect
		// Redis server errors
		FailOpen: false,
	})

	// make sure we fill the buckets if we are running the exmaple back to back
	time.Sleep(2 * interval)

	// status:
	// key1: 2 tokens
	// key2: 2 tokens

	// use 2 key1 tokens and 1 key2 token
	fmt.Printf("l.AllowN(key1, 2):\ttrue == %v\n", l.AllowN(key1, 2))
	fmt.Printf("l.Allow(key2):\t\ttrue == %v\n", l.Allow(key2))

	// status:
	// key1: 0 tokens
	// key2: 1 tokens

	// replenish 1 token
	time.Sleep(interval)

	// status:
	// key1: 1 tokens
	// key2: 2 tokens

	// use 1 key1 token and 2 key2 tokens
	fmt.Printf("l.Allow(key1):\t\ttrue == %v\n", l.Allow(key1))
	fmt.Printf("l.AllowN(key2, 2):\ttrue == %v\n", l.AllowN(key2, 2))

	// status:
	// key1: 0 tokens
	// key2: 0 tokens

	// try to allow from both empty buckets
	fmt.Printf("l.Allow(key1):\t\tfalse == %v\n", l.Allow(key1))
	fmt.Printf("l.Allow(key2):\t\tfalse == %v\n", l.Allow(key2))
}
