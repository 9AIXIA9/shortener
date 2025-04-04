package main

import (
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"github.com/zeromicro/go-zero/core/logx"
	"os"
	"shortener/internal/config"
	"shortener/internal/handler"
	"shortener/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/shortener-api.yaml", "the config file")

func main() {
	flag.Parse()

	//加载环境变量
	loadEnv()

	//加载配置（自动替换环境变量）
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	server := rest.MustNewServer(c.RestConf)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}

func loadEnv() {
	//加载根目录的 .env（覆盖现有变量）
	if err := godotenv.Overload(); err != nil {
		logx.Severef("load .env failed,err: %v", err)
	}

	//获取并设置环境变量 APP_ENV
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
		logx.Infof("APP_ENV not set, defaulting to: %s", env)
	}

	//加载环境专属的 .env 文件（如 .env.dev）
	envFile := fmt.Sprintf(".env.%s", env)
	if err := godotenv.Overload(envFile); err != nil {
		logx.Severef("failed to load environment file %s: %v", envFile, err)
	}
}
