package main

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisDB struct {
	client *redis.Client
}

// newRedisDB creates new RedisDB.
func newRedisDB(connectURL string) (*RedisDB, error) {
	opt, err := redis.ParseURL(connectURL)
	if err != nil {
		return nil, err
	}
	rdb := &RedisDB{
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

// set, depending on key and its value, creates, updates or does nothing with the key and value.
// Value is always timestamp when key was created or updated. Format in RFC3339.
// If key does not exist key is created with value is current timestamp.
// If key exists and key has not expired, nothing happens.
// If key exists and key has expired, value of key is updated to current timestamp.
func (rdb *RedisDB) set(key string) (redisChangeType, error) {
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
