package main

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type redisDB struct {
	client *redis.Client
}

func newRedisDB(connectURL string) (*redisDB, error) {
	opt, err := redis.ParseURL(connectURL)
	if err != nil {
		return nil, err
	}
	rdb := &redisDB{
		client: redis.NewClient(opt),
	}
	return rdb, nil
}

type redisChangeType int

const (
	redisChangeAdd redisChangeType = iota
	redisChangeUpdate
	redisChangeNothing
	redisChangeErr
)

func (rdb *redisDB) set(key string) (redisChangeType, error) {
	tsNow := time.Now()
	tsNowRFC3339 := tsNow.Format(time.RFC3339)
	ctx := context.Background()

	val, err := rdb.client.Get(ctx, key).Result()

	if err == redis.Nil { // key does not exist
		if err = rdb.client.Set(ctx, key, tsNowRFC3339, 0).Err(); err != nil {
			return redisChangeErr, err
		}
		return redisChangeAdd, nil
	}
	if err != nil {
		return redisChangeErr, err
	}

	tsFromDB, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return redisChangeErr, err
	}

	if tsNow.Sub(tsFromDB) > redisExpiration { // key expired
		if err = rdb.client.Set(ctx, key, tsNowRFC3339, 0).Err(); err != nil {
			return redisChangeErr, err
		}
		return redisChangeUpdate, nil
	}

	return redisChangeNothing, nil
}
