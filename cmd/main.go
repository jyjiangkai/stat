package main

import (
	"fmt"
	"io"
	"os"

	"github.com/gin-contrib/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"

	"github.com/jyjiangkai/stat/config"
	"github.com/jyjiangkai/stat/controller"
	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/mailchimp"
	"github.com/jyjiangkai/stat/monitor"
	"github.com/jyjiangkai/stat/router"
	"github.com/jyjiangkai/stat/utils"
)

type Config struct {
	Port      int              `yaml:"port"`
	DB        db.Config        `yaml:"mongodb"`
	Monitor   monitor.Config   `yaml:"monitor"`
	MailChimp mailchimp.Config `yaml:"mailchimp"`
	S3        config.S3        `yaml:"s3"`
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

	monitor.Init(ctx, cfg.Monitor)
	mailchimp.Init(ctx, cfg.MailChimp)

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

	downloadService := services.NewDownloadService(cfg.S3)
	router.RegisterDownloadRouter(
		e.Group("/download"),
		controller.NewDownloadController(downloadService),
	)

	actionService := services.NewActionService(cli)
	if err = actionService.Start(); err != nil {
		panic("failed to start action service: " + err.Error())
	}
	router.RegisterActionsRouter(
		e.Group("/actions"),
		controller.NewActionController(actionService),
	)

	statService := services.NewStatService(cli)
	if err = statService.Start(); err != nil {
		panic("failed to start stat service: " + err.Error())
	}

	trackService := services.NewTrackService(cli)
	if err = trackService.Start(); err != nil {
		panic("failed to start track service: " + err.Error())
	}

	alarmService := services.NewAlarmService(cli)
	if err = alarmService.Start(); err != nil {
		panic("failed to start alarm service: " + err.Error())
	}

	cohortService := services.NewCohortService(cli)
	if err = cohortService.Start(); err != nil {
		panic("failed to start cohort service: " + err.Error())
	}

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
