package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/monitor"
)

var (
	Database string = "vanus-cloud-prod"
	Username string = "vanus-cloud-prod-rw"
	Password string = "K1Cr0WGQca396QLu"
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

	monitorCfg := monitor.Config{
		Enable:     true,
		WebhookUrl: "https://9vew3ud8zabfjpgh.connector.vanustest.com/api/v1/source/http/6503ca00776573a0df3ba173",
	}
	monitor.Init(ctx, monitorCfg)

	now := time.Now()
	svc := services.NewAlarmService(cli)
	if err := svc.AlarmOfUsage(ctx, now); err != nil {
		log.Error(ctx).Err(err).Msg("failed to execute alarm service")
		return
	}
	log.Info(ctx).Msg("success to execute alarm service")
}
