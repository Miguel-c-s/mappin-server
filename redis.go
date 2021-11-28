package main

import (
	"context"
	"os"
	"time"

	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

var tokensClient *redis.Client = RedisConnect("6379")

var codesClient *redis.Client = RedisConnect("6380")

/*RedisConnect - connects to redis
*
*
*
 */
func RedisConnect(port string) *redis.Client {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Initializing Redis client
	dsn := os.Getenv("REDIS_DSN")
	if len(dsn) == 0 {
		dsn = "127.0.0.1:" + port
	}
	redisClientLocal := redis.NewClient(&redis.Options{
		Addr: dsn,
	})
	_, err := redisClientLocal.Ping(ctx).Result()
	if err != nil {
		log.Fatal(err)
		panic(err)
	}
	return redisClientLocal
}
