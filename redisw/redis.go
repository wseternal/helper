// redis wrapper
package redisw

import (
	"fmt"
	"github.com/go-redis/redis"
	"github.com/wseternal/helper"
	"strconv"
	"time"
)

type Client struct {
	*redis.Client
	Opt  *redis.Options
	FOpt *redis.FailoverOptions
}

type TimeRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

var (
	DefaultOption = &redis.Options{
		Addr:         "127.0.0.1:6379",
		DialTimeout:  2 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		MaxRetries:   3,
	}

	DefaultFailoverOption = &redis.FailoverOptions{
		MasterName:    "mymaster",
		SentinelAddrs: []string{":26379"},
		DialTimeout:   2 * time.Second,
		ReadTimeout:   3 * time.Second,
		WriteTimeout:  3 * time.Second,
		MaxRetries:    3,
	}
	AllZRange = &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}
)

func NewFailoverClient(opt *redis.FailoverOptions) (*Client, error) {
	var err error

	if opt == nil {
		opt = DefaultFailoverOption
	}
	db := redis.NewFailoverClient(opt)
	if err = db.Ping().Err(); err != nil {
		return nil, err
	}
	return &Client{
		Client: db,
		FOpt:   opt,
	}, nil
}

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
		Opt:    opt,
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
	if val, err = cmd.Int64(); err != nil {
		return dfl
	}
	return val
}

// convert TimeRange to redis ZRangeBy
func (p *TimeRange) ToZRangeScoreOpt() (opt *redis.ZRangeBy, err error) {
	scoreOpt := &redis.ZRangeBy{
		Min: strconv.FormatInt(helper.UnixDate(0), 10),
		Max: "+inf",
	}
	if p.Start > 0 {
		scoreOpt.Min = strconv.Itoa(p.Start)
		if p.End >= p.Start {
			scoreOpt.Max = strconv.Itoa(p.End)
		} else {
			return nil, fmt.Errorf("invalid end: %d, less that start %d", p.End, p.Start)
		}
	} else if p.Start < 0 {
		scoreOpt.Min = strconv.FormatInt(helper.UnixDate(p.Start), 10)
		if p.End >= p.Start {
			scoreOpt.Max = strconv.FormatInt(helper.UnixDate(p.End), 10)
		} else {
			return nil, fmt.Errorf("invalid end: %d, less that start %d", p.End, p.Start)
		}
	}
	return scoreOpt, nil
}
