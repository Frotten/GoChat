package redis

import (
	"GopherAI/config"
	"context"
	"errors"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

var Rdb *redis.Client

var ctx = context.Background()

func Init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.RedisHost
	port := conf.RedisConfig.RedisPort
	password := conf.RedisConfig.RedisPassword
	db := conf.RedisDb
	addr := host + ":" + strconv.Itoa(port)

	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := Rdb.Ping(ctx).Err(); err != nil {
		log.Printf("[redis] 连接失败 %s: %v（验证码等功能将不可用，请检查 config.toml 中 redisConfig）", addr, err)
	} else {
		log.Printf("[redis] 连接成功 %s", addr)
	}
}

func SetCaptchaForEmail(email, captcha string) error {
	key := GenerateCaptcha(email)
	expire := 2 * time.Minute
	return Rdb.Set(ctx, key, captcha, expire).Err()
}

func CheckCaptchaForEmail(email, userInput string) (bool, error) {
	key := GenerateCaptcha(email)
	storedCaptcha, err := Rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil
		}
		return false, err
	}
	if strings.EqualFold(storedCaptcha, userInput) {
		_ = Rdb.Del(ctx, key).Err()
		return true, nil
	}
	return false, nil
}
