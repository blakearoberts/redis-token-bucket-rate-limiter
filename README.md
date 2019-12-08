# redis-token-bucket-rate-limiter

A library which utilizes Redis to implement distributed token bucket rate limiting.

## Token Bucket Rate Limiting

Consider a rate limiter configured with a rate limit of 10 queries per second (qps) and a burst allowance of 20. The rate limiter has a bucket size of 20 that replenishes at a rate of 10 tokens per second. If at `T0` 20 queries are requested, the rate limiter will allow all 20 queries. The rate limiter will not allow additional queries until `T1`. At `T1` the rate limiter's bucket will be given an allowance of 10 tokens. At `T3` if no queries have been made, the rate limiter will be given an additional allowance of 10 tokens capping its bucket at 20 tokens. If by `T4` no queries have been made, the token bucket will not recieve any additional tokens as it is at its burst limit.

## Quick Setup

```go
package main

import (
    "fmt"
    "time"

    "github.com/blakearoberts/redis-token-bucket-rate-limiter/limiter"
)

func main() {
    l := limiter.New(limiter.Config{
        Type: limiter.TypeRedis, // use redis as apposed to in-memory/disabled limiters
        Address: ":6379",        // redis server address
        RateLimit: 10.0,         // measured in queries per "Interval"
        BurstLimit: 20,          // size of the token bucket refilled at "RateLimit" tokens per "Interval"
        Interval: time.Second,   // the interval of the rate limiter
        FailOpen: true,          // allow queries when a redis server error is encountered
    })

    key := "foo"
    if !l.Allow(key) {
        fmt.Printf("%s is not allowed to do stuff right now\n", key)
    }
}
```

## Example

Check out the [example](./example/main.go) for more information.

To run the example, make sure you have a Redis server running locally at `:6379`, then run the make directive:

```bash
$ make example
go run example/main.go
l.AllowN(key1, 2):  true == true
l.Allow(key2):      true == true
l.Allow(key1):      true == true
l.AllowN(key2, 2):  true == true
l.Allow(key1):      false == false
l.Allow(key2):      false == false
```

## AllowDynamic

Because rate limiters are not preconfigured (when the system is presented a new key, a rate limiter is created on the fly), configurable, per-key, rate and burst limits are accomplished by `AllowDynamic` and `AllowNDynamic`:

```go
package main

import (
    "fmt"

    "github.com/blakearoberts/redis-token-bucket-rate-limiter/limiter"
)

const burstLimit = 500

var rateLimits = map[string]float64{}{
    "account1": 100.0,
    "account2": 300.0,
}

func main() {
    l := limiter.New(limiter.Config{
        Type: limiter.TypeRedis,
        RateLimit: 0.0, // disallow keys not defined in the rateLimits map
        BurstLimit: burstLimit,
        Address: ":6379",
        FailOpen: true,
    })

    key := "account1"
    if !l.AllowDynamic(key, rateLimits[key], burstLimit) {
        fmt.Printf("%s is not allowed to do stuff right now\n", key)
    }
}
```

## Rate Limit Intervals

A `Limiter` defaults to 1 second rate limit intervals. This means that, the if the rate limit has a value of `10.0`, a token bucket will be replinished at 10 tokens per second. This can be increased or decreased to any `time.Duration`. It works by truncating times returned by `time.Now()`. The following Go program demonstrates how the trunctation works:

```go
package main

import (
    "fmt"
    "time"
)

func main() {
    now := time.Now().UTC()

    fmt.Printf("untracated:\t%v\n",
        now.Format(time.RFC3339Nano))

    fmt.Printf("nearest 100 us:\t%v\n",
        now.Truncate(100*time.Microsecond).Format(time.RFC3339Nano))

    fmt.Printf("nearest 500 ms:\t%v\n",
        now.Truncate(500*time.Millisecond).Format(time.RFC3339Nano))

    fmt.Printf("nearest 1 sec:\t%v\n",
        now.Truncate(time.Second).Format(time.RFC3339Nano))

    fmt.Printf("nearest 30 min:\t%v\n",
        now.Truncate(30*time.Minute).Format(time.RFC3339Nano))
}
```

```bash
$ go run main.go
untracated:     2019-12-07T21:30:45.831236Z
nearest 100 us: 2019-12-07T21:30:45.8312Z
nearest 500 ms: 2019-12-07T21:30:45.5Z
nearest 1 sec:  2019-12-07T21:30:45Z
nearest 30 min: 2019-12-07T21:30:00Z
```

## Local Development and Testing

Use `limiter.TypeInMemory` when a Redis server is not available:

```go
l := limiter.New(limiter.Config{
    Type: limiter.TypeInMemory,
    RateLimit: 10.0,
    BurstLimit: 20,
})
```

Use `limiter.TypeDisabled` when unit testing or perhaps load testing:

```go
l := limiter.New(limiter.Config{Type: limiter.TypeDisabled})
```
