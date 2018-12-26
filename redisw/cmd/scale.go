package main

import (
	"bitbucket.org/wseternal/helper/codec"
	"bitbucket.org/wseternal/helper/redisw"
	"fmt"
	"github.com/go-redis/redis"
)

type MeasureResult struct {
	Weight float32 `json:"weight"`
	Height float32 `json:"height"`
	Bmi float32 `json:"bmi"`
	Scale string `json:"scale"`
	Timestamp int64 `json:"ts"`
}

type FollowDetail struct {
	UnionID string `json:"unionid"`
	AppID string `json:"appid"`
	OpenID string `json:"openid"`
}

func addUnfollow(c *redisw.Client, unionid, appid string, tsUnfollow int64) error {
	var key string
	var err error

	key = fmt.Sprintf("unfollow:user:%s", unionid)

	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsUnfollow),
		Member: codec.JsonMarshal(struct {
			AppID string `json:"appid"`
			Timestamp int64 `json:"ts"`
		} {
			AppID:appid,
			Timestamp: tsUnfollow,
		}),
	}).Err(); err != nil {
		return err
	}

	key = "unfollow:all"
	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsUnfollow),
		Member: codec.JsonMarshal(struct {
			UnionID string `json:"unionid"`
			AppID string `json:"appid"`
			Timestamp int64 `json:"ts"`
		} {
			UnionID: unionid,
			AppID:appid,
			Timestamp: tsUnfollow,
		}),
	}).Err(); err != nil {
		return err
	}

	return nil
}

func addFollow(c *redisw.Client, scale, unionid, appid, openid string, tsFollow int64) error {
	var key string
	var err error

	key = "follow:all"
	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsFollow),
		Member: codec.JsonMarshal(struct {
			Scale string `json:"scale"`
			UnionID string `json:"unionid"`
			AppID string `json:"appid"`
			Timestamp int64 `json:"ts"`
		} {
			Scale: scale,
			UnionID: unionid,
			AppID:appid,
			Timestamp: tsFollow,
		}),
	}).Err(); err != nil {
		return err
	}

	key = "followset:all"
	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsFollow),
		Member: codec.JsonMarshal(struct {
			UnionID string `json:"unionid"`
			AppID string `json:"appid"`
		} {
			UnionID: unionid,
			AppID: appid,
		}),
	}).Err(); err != nil {
		return err
	}

	key = fmt.Sprintf("follow:user:%s", unionid)

	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsFollow),
		Member: codec.JsonMarshal(struct {
			Scale string `json:"scale"`
			AppID string `json:"appid"`
			Timestamp int64 `json:"ts"`
		} {
			Scale: scale,
			AppID:appid,
			Timestamp: tsFollow,
		}),
	}).Err(); err != nil {
		return err
	}

	key = fmt.Sprintf("openid:user:%s", unionid)

	if err = c.HSet(key, appid, openid).Err(); err != nil {
		return err
	}

	key = fmt.Sprintf("followset:user:%s", unionid)
	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsFollow),
		Member: appid,
	}).Err(); err != nil {
		return err
	}

	key = fmt.Sprintf("followset:appid:%s", appid)
	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsFollow),
		Member: unionid,
	}).Err(); err != nil {
		return err
	}

	key = fmt.Sprintf("followset:scale:%s", scale)
	if err = c.ZAdd(key, redis.Z{
		Score: float64(tsFollow),
		Member: codec.JsonMarshal(struct {
			UnionID string `json:"unionid"`
			AppID string `json:"appid"`
		} {
			UnionID: unionid,
			AppID: appid,
		}),
	}).Err(); err != nil {
		return err
	}

	return nil
}


func addUserMeasureResult(c *redisw.Client, unionid string, r *MeasureResult) error {
	key := fmt.Sprintf("measure:user:%s", unionid)
	var err error

	if err = c.ZAdd(key, redis.Z{
		Score: float64(r.Timestamp),
		Member: codec.JsonMarshal(r),
	}).Err(); err != nil {
		return err
	}

	key = fmt.Sprintf("measure:scale:%s", r.Scale)
	var mres = struct {
		UnionID string `json:"unionid"`
		Timestamp int64 `json:"ts"`
	} {
		UnionID: unionid,
		Timestamp: r.Timestamp,
	}
	if err = c.ZAdd(key, redis.Z{
		Score: float64(r.Timestamp),
		Member: codec.JsonMarshal(mres),
	}).Err(); err != nil {
		return err
	}

	key = "measure:all"
	var ares = struct {
		Scale string `json:"scale"`
		Timestamp int64 `json:"ts"`
	}  {
		Scale: r.Scale,
		Timestamp: r.Timestamp,
	}
	if err = c.ZAdd(key, redis.Z{
		Score: float64(r.Timestamp),
		Member: codec.JsonMarshal(ares),
	}).Err(); err != nil {
		return err
	}

	return nil
}

func IsPublicAccount(id string) bool {
	if len(id) == 18 && id[0:2] == "wx" {
		return true
	}
	return false
}
