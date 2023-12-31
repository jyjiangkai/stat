package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
)

var (
	Database string = "vanus-cloud-prod"
	Username string = "vanus-cloud-prod-rw"
	Password string = ""
	Address  string = "cluster1.odfrc.mongodb.net"
)

func main() {
	ctx := context.Background()
	cfg := db.Config{
		Database: Database,
		Address:  Address,
		Username: Username,
		Password: Password,
	}
	cli, err := db.Init(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize mongodb client: %s", err))
	}
	defer func() {
		_ = cli.Disconnect(ctx)
	}()

	svc := services.NewCohortService(cli)
	err = svc.WeeklyCohortAnalysis(ctx, time.Now())
	if err != nil {
		log.Error(ctx).Err(err).Msg("weekly cohort analysis failed")
		return
	}
	log.Info(ctx).Msg("finish weekly cohort analysis")
}
