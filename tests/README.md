# Integration Tests

Integration tests have been written to test functionality against a live Redis server. To run the tests locally, perform the following setup and execution:

## Setup

1. Install [Redis](https://redis.io/topics/quickstart)

    ```bash
    brew install redis
    ```

1. Start Redis

    ```text
    $ redis-server
    4989:C 08 Dec 2019 13:17:41.964 # oO0OoO0OoO0Oo Redis is starting oO0OoO0OoO0Oo
    4989:C 08 Dec 2019 13:17:41.965 # Redis version=5.0.6, bits=64, commit=00000000, modified=0, pid=4989, just started
    4989:C 08 Dec 2019 13:17:41.965 # Warning: no config file specified, using the default config. In order to specify a config file use redis-server /path/to/redis.conf
    4989:M 08 Dec 2019 13:17:41.966 * Increased maximum number of open files to 10032 (it was originally set to 256).
                   _._
              _.-``__ ''-._
         _.-``    `.  `_.  ''-._           Redis 5.0.6 (00000000/0) 64 bit
     .-`` .-```.  ```\/    _.,_ ''-._
    (    '      ,       .-`  | `,    )     Running in standalone mode
    |`-._`-...-` __...-.``-._|'` _.-'|     Port: 6379
    |    `-._   `._    /     _.-'    |     PID: 4989
    `-._    `-._  `-./  _.-'      _.-'
    |`-._`-._    `-.__.-'    _.-'_.-'|
    |    `-._`-._        _.-'_.-'    |           http://redis.io
    `-._    `-._`-.__.-'_.-'      _.-'
    |`-._`-._    `-.__.-'    _.-'_.-'|
    |    `-._`-._        _.-'_.-'    |
     `-._    `-._`-.__.-'_.-'    _.-'
         `-._    `-.__.-'    _.-'
             `-._        _.-'
                 `-.__.-'

    4989:M 08 Dec 2019 13:17:41.970 # Server initialized
    4989:M 08 Dec 2019 13:17:41.971 * Ready to accept connections
    ```

## Exectuion

The following will read/write to the locally running Redis server. Specifically, before any assertions are made, the database is cleared.

```bash
$ make integration
go test ./tests -count=1
ok      github.com/blakearoberts/redis-token-bucket-rate-limiter/tests  8.018s
```
