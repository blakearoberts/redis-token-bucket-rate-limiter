package limiter

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/stretchr/testify/mock"
)

type mockConn struct {
	mock.Mock
}

func (m *mockConn) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockConn) Err() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockConn) Do(cmd string, cmdArgs ...interface{}) (interface{}, error) {
	args := m.Called(cmd, cmdArgs)
	return args.Get(0), args.Error(1)
}

func (m *mockConn) Send(cmd string, cmdArgs ...interface{}) error {
	args := m.Called(cmd, cmdArgs)
	return args.Error(0)
}

func (m *mockConn) Flush() error {
	args := m.Called()
	return args.Error(0)
}

func (m *mockConn) Receive() (interface{}, error) {
	args := m.Called()
	return args.Get(0), args.Error(1)
}

func newMockRedisLimiter(m *mockConn) *redisLimiter {
	l := New(Config{
		Type:       TypeRedis,
		RateLimit:  10,
		BurstLimit: 20,
	}).(*redisLimiter)

	l.pool.Dial = func() (redis.Conn, error) {
		return m, nil
	}
	var n []interface{} = nil
	m.On("Do", "", n).Return(nil, nil).Once()
	m.On("Err").Return(nil).Once()
	m.On("Close").Return(nil).Once()
	return l
}

func TestAllowNoKey(t *testing.T) {
	m := &mockConn{}
	l := newMockRedisLimiter(m)
	key := "foo"

	m.On(
		"Do", "LRANGE", []interface{}{key, 0, 1},
	).Return([]interface{}{}, nil).Once()

	m.On(
		"Do",
		mock.MatchedBy(func(cmd string) bool {
			return cmd == "LPUSH"
		}),
		mock.MatchedBy(func(args []interface{}) bool {
			if len(args) != 3 {
				return false
			}
			if args[0].(string) != key {
				return false
			}
			if args[1].(float64) != float64(l.burst-1) {
				return false
			}
			_, ok := args[2].(int64)
			return ok
		}),
	).Return(int64(2), nil).Once()

	if !l.Allow(key) {
		t.Errorf("expected to allow key: %s", key)
	}
}

func TestAllowAddTokens(t *testing.T) {
	m := &mockConn{}
	l := newMockRedisLimiter(m)
	key := "foo"

	m.On("Do", "LRANGE", []interface{}{key, 0, 1}).Return(
		[]interface{}{
			[]byte{'0'},
			[]byte{'0'},
		}, nil,
	).Once()

	var n []interface{} = nil
	m.On("Send", "MULTI", n).Return(nil).Once()
	m.On(
		"Send", "LSET", []interface{}{key, 0, float64(l.burst - 2)},
	).Return(nil, nil).Once()
	m.On(
		"Send", "LSET",
		[]interface{}{key, 1, time.Now().Round(time.Second).Unix()},
	).Return(nil, nil).Once()
	m.On("Do", "EXEC", n).Return(nil, nil).Once()

	if !l.AllowN(key, 2) {
		t.Errorf("expected to allow key: %s", key)
	}
}

func TestAllowNoTokens(t *testing.T) {
	m := &mockConn{}
	l := newMockRedisLimiter(m)
	key := "foo"

	m.On("Do", "LRANGE", []interface{}{key, 0, 1}).Return(
		[]interface{}{
			[]byte{'0'},
			[]byte(fmt.Sprintf("%d", time.Now().Round(time.Second).Unix())),
		}, nil,
	).Once()

	if l.AllowDynamic(key, 10.0, 20) {
		t.Errorf("expected to not allow key: %s", key)
	}
}

func TestRedisLRangeError(t *testing.T) {
	m := &mockConn{}
	l := newMockRedisLimiter(m)
	key := "foo"

	m.On("Do", "LRANGE", []interface{}{key, 0, 1}).Return(
		nil, errors.New("not good"),
	).Once()

	if l.AllowNDynamic(key, 1, 10.0, 20) {
		t.Errorf("expected to not allow key: %s", key)
	}
}

func TestRedisLPushError(t *testing.T) {
	m := &mockConn{}
	l := newMockRedisLimiter(m)
	key := "foo"

	m.On(
		"Do", "LRANGE", []interface{}{key, 0, 1},
	).Return([]interface{}{}, nil).Once()

	m.On(
		"Do",
		mock.MatchedBy(func(cmd string) bool {
			return cmd == "LPUSH"
		}),
		mock.MatchedBy(func(args []interface{}) bool {
			if len(args) != 3 {
				return false
			}
			if args[0].(string) != key {
				return false
			}
			if args[1].(float64) != float64(l.burst-1) {
				return false
			}
			_, ok := args[2].(int64)
			return ok
		}),
	).Return(int64(0), errors.New("not good")).Once()

	if l.Allow(key) {
		t.Errorf("expected to not allow key: %s", key)
	}
}

func TestRedisScanError(t *testing.T) {
	m := &mockConn{}
	l := newMockRedisLimiter(m)
	key := "foo"

	m.On("Do", "LRANGE", []interface{}{key, 0, 1}).Return(
		[]interface{}{
			[]byte{'h'},
			[]byte{'i'},
		}, nil,
	).Once()

	if l.Allow(key) {
		t.Errorf("expected to not allow key: %s", key)
	}
}

func TestRedisExecError(t *testing.T) {
	m := &mockConn{}
	l := newMockRedisLimiter(m)
	key := "foo"

	m.On("Do", "LRANGE", []interface{}{key, 0, 1}).Return(
		[]interface{}{
			[]byte{'0'},
			[]byte{'0'},
		}, nil,
	).Once()

	var n []interface{} = nil
	m.On("Send", "MULTI", n).Return(nil).Once()
	m.On(
		"Send", "LSET", []interface{}{key, 0, float64(l.burst - 1)},
	).Return(nil, nil).Once()
	m.On(
		"Send", "LSET",
		[]interface{}{key, 1, time.Now().Round(time.Second).Unix()},
	).Return(nil, nil).Once()
	m.On("Do", "EXEC", n).Return(nil, errors.New("not good")).Once()

	if l.Allow(key) {
		t.Errorf("expected to not allow key: %s", key)
	}
}

func TestBadLimiterType(t *testing.T) {
	l := New(Config{
		Type: -1,
	})
	if l != nil {
		t.Error("expected limiter to be nil when given a bad type")
	}
}

func TestInMemoryLimiter(t *testing.T) {
	l := New(Config{
		Type:       TypeInMemory,
		RateLimit:  1.0,
		BurstLimit: 8,
	})
	key := "foo"

	if !l.Allow(key) {
		t.Errorf("expected to allow key: %s", key)
	}
	if !l.AllowN(key, 2) {
		t.Errorf("expected to allow key: %s", key)
	}
	if !l.AllowDynamic(key, 0.0, 8) {
		t.Errorf("expected to allow key: %s", key)
	}
	if l.AllowNDynamic(key, 2, 0.0, 0) {
		t.Errorf("expected to allow key: %s", key)
	}
}

func TestDisabledLimiter(t *testing.T) {
	l := New(Config{
		Type: TypeDisabled,
	})
	if !l.Allow("") {
		t.Error("expected disabled limiter to allow")
	}
	if !l.AllowN("", 1) {
		t.Error("expected disabled limiter to allow")
	}
	if !l.AllowDynamic("", 0, 0) {
		t.Error("expected disabled limiter to allow")
	}
	if !l.AllowNDynamic("", 0, 0, 0) {
		t.Error("expected disabled limiter to allow")
	}
}
