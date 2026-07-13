package services

import (
	"context"
	"log"
	"os"

	"github.com/redis/go-redis/v9"
)

var Rdb *redis.Client
var Ctx = context.Background()

func RedisConnect() {
	redisUrl := os.Getenv("REDIS_URL")
	opt, err := redis.ParseURL(redisUrl)
	if err != nil {
		log.Fatal("Error parsing Redis URL:", err)
	}
	Rdb = redis.NewClient(opt)

	err = Rdb.Get(Ctx, "foo").Err()
	if err != nil {
		log.Fatal("Error connecting to Redis:", err)
	}
	log.Println("Connected to Redis successfully")
}
