package main

import (
	"flag"
	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest"
	"os"
	"os/signal"
	"shortener/internal/config"
	"shortener/internal/errorhandler"
	"shortener/internal/types/errorx"
	"syscall"
)

var configFile = flag.String("f", "etc/shortener-api.yaml", "the config file")

func main() {
	flag.Parse()

	//加载环境变量
	config.LoadEnv()

	//加载配置（自动替换环境变量）
	var c config.Config
	conf.MustLoad(*configFile, &c, conf.UseEnv())

	// 使用配置初始化错误处理器
	errorhandler.Init(c.ErrorHandler, func(err *errorx.ErrorX) error {
		logx.Error(err)
		logx.ErrorStack()
		return nil
	}) // 添加消费者

	server := rest.MustNewServer(c.RestConf)

	// 创建停止信号通道
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// 非阻塞方式启动服务
	go func() {
		logx.Infof("Starting server at %s:%d...\n", c.Host, c.Port)
		server.Start()
	}()

	// 等待停止信号
	<-stopChan
	logx.Info("A stop signal is received and graceful closure begins...")

	// 按顺序关闭组件
	server.Stop()
	errorhandler.Shutdown()
	logx.Info("the service is completely shut down")
}
