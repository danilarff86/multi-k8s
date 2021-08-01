package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
)

const (
	REDIS_HOST = "REDIS_HOST"
	REDIS_PORT = "REDIS_PORT"
)

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.Println("starting Golang worker")
	ctx := context.Background()

	redisClient, err := connectToRedis(ctx)
	checkErr(err)
	defer redisClient.Close()

	pubsub := redisClient.PSubscribe(ctx, "insert")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		log.Printf("new message: %s", msg.Payload)

		index, err := strconv.Atoi(msg.Payload)
		if err != nil {
			log.Printf("unable to parse integer: %s", msg.Payload)
			continue
		}

		val := fib(index)
		err = redisClient.HSet(ctx, "values", msg.Payload, val).Err()
		if err != nil {
			log.Printf("unable to set value '%d' for key: '%s'", val, msg.Payload)
			continue
		}
	}
}

func fib(index int) int {
	if index < 2 {
		return 1
	}
	return fib(index-1) + fib(index-2)
}

func getEnv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		panic(fmt.Sprintf("environment variable '%s' is not set", name))
	}
	return val
}

func connectToRedis(ctx context.Context) (redis.UniversalClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", getEnv(REDIS_HOST), getEnv(REDIS_PORT)),
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
