// redis wrapper
package redisw

import (
	"github.com/go-redis/redis"
	"time"
)

type Client struct {
	*redis.Client
}

var (
	DefaultOption = &redis.Options{
		Addr: "127.0.0.1:6379",
		MaxRetries: 0,
		DialTimeout: 2*time.Second,
		ReadTimeout: 3*time.Second,
	}
)

// NewClient create new redis client, use ping to check
// use Default option is opt is nil
func NewClient(opt *redis.Options) (*Client, error) {
	var err error
	if opt == nil {
		opt = DefaultOption
	}
	db := redis.NewClient(opt)
	if err = db.Ping().Err(); err != nil {
		return nil, err
	}
	return &Client{
		Client: db,
	}, nil
}

// GetInt64 if errStr is non-nil, any error occurred will be recorded
func (c *Client) GetInt64(key string, dfl int64, errStr *string) int64 {
	var err error
	defer func() {
		if err != nil && errStr != nil {
			*errStr = err.Error()
		}
	}()

	cmd := c.Client.Get(key)
	if err = cmd.Err(); err != nil {
		return dfl
	}

	var val int64
	if val, err =  cmd.Int64(); err != nil {
		return dfl
	}
	return val
}
