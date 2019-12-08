package main

import (
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/blakearoberts/redis-token-bucket-rate-limiter/limiter"
)

const (
	address  = ":6379"
	rate     = 1.0
	burst    = 2
	interval = 2 * time.Second
	key      = "foo"
)

func Test(t *testing.T) {
	// get database connection
	c, err := redis.Dial("tcp", address)
	defer c.Close()
	if err != nil {
		t.Fatal(err)
	}

	// clear database
	if err := c.Send("FLUSHALL"); err != nil {
		t.Fatal(err)
	}
	c.Flush()

	// setup limiter
	l := limiter.New(limiter.Config{
		Type:       limiter.TypeRedis,
		Address:    address,
		RateLimit:  rate,
		BurstLimit: burst,
		Interval:   interval,
		FailOpen:   false,
	})

	// test using a single token on a new key
	if !l.Allow(key) {
		t.Fatal("did not allow initial key")
	}
	tokens, _ := getKey(c, key)
	if tokens != float64(burst-1) {
		t.Fatalf("expected %v tokens: %v", float64(burst-1), tokens)
	}

	// fill the bucket
	time.Sleep(rate * burst * interval)

	// test using all the tokens at once
	if !l.AllowN(key, burst) {
		t.Fatal("did not allow burst of 2")
	}
	tokens, _ = getKey(c, key)
	if tokens != 0 {
		t.Fatalf("expected 0 tokens: %v", tokens)
	}

	// make sure we get rate limited
	if l.Allow(key) {
		t.Fatal("did not rate limit empty bucket")
	}

	// fill the bucket
	time.Sleep(rate * burst * interval)

	// use all but one token
	if !l.AllowN(key, burst-1) {
		t.Fatal("did not allow key")
	}
	tokens, _ = getKey(c, key)
	if tokens != 1 {
		t.Fatalf("expected 1 tokens: %v", tokens)
	}
}

func getKey(c redis.Conn, key string) (tokens float64, last int64) {
	resp, _ := redis.Values(c.Do("LRANGE", key, 0, 1))
	redis.Scan(resp, &tokens, &last)
	return
}
