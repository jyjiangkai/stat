package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/jyjiangkai/stat/controller"
	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/router"
	"github.com/jyjiangkai/stat/utils"
)

type Config struct {
	Port int       `yaml:"port"`
	DB   db.Config `yaml:"mongodb"`
}

var (
	ConfigFile = "config/server.yaml"
)

func main() {
	ctx := utils.SetupSignalContext() // use system signal
	f, err := os.Open(ConfigFile)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to open config file")
		os.Exit(1)
	}

	data, err := io.ReadAll(f)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to read config file")
		os.Exit(1)
	}

	cfg := Config{}
	if err = yaml.Unmarshal(data, &cfg); err != nil {
		log.Error(ctx).Err(err).Msg("failed to unmarshal config file")
		os.Exit(1)
	}

	cli, err := db.Init(ctx, cfg.DB)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize mongodb client: %s", err))
	}
	defer func() {
		_ = cli.Disconnect(ctx)
	}()

	lg := logger.SetLogger(
		logger.WithLogger(log.CustomLogger),
	)

	eng := gin.New()
	eng.Use(
		gin.Recovery(),
		requestid.New(),
	)

	e := eng.Group("/v1")
	e.Use(
		lg,
	)

	userService := services.NewUserService(cli)
	if err = userService.Start(); err != nil {
		panic("failed to start user service: " + err.Error())
	}
	router.RegisterUsersRouter(
		e.Group("/users"),
		controller.NewUserController(userService),
	)

	// refreshService := services.NewRefreshService(cli)
	// if err = refreshService.Start(); err != nil {
	// 	panic("failed to start refresh service: " + err.Error())
	// }

	// router.RegisterCollectRouter(
	// 	e.Group("/collect"),
	// 	controller.NewCollectorController(collectorService),
	// )

	go func() {
		if err = eng.Run(fmt.Sprintf("0.0.0.0:%d", cfg.Port)); err != nil {
			panic(fmt.Sprintf("failed to start HTTP server: %s", err))
		}
	}()

	select {
	case <-ctx.Done():
		log.Info(ctx).Msg("received system signal, preparing exit")
	}
	if err = userService.Stop(); err != nil {
		log.Warn(ctx).Err(err).Msg("error when stop UserService")
	}
	log.Info(ctx).Msg("the Vanus Stat Server has been shutdown gracefully")
}
