package config

import "github.com/zeromicro/go-zero/rest"

type Config struct {
	rest.RestConf
	Redis RedisConfig
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}
