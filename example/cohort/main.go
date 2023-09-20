package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models/cloud"
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

	base := cloud.NewBase(ctx)
	base.CreatedAt = time.Date(2023, 3, 9, 0, 0, 0, 0, time.UTC)
	user := &cloud.User{
		Base: base,
		OID:  "github|10882129",
	}
	svc := services.NewRefreshService(cli)
	cohort, err := svc.GetCohort(ctx, user)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to get user cohort")
		return
	}
	log.Info(ctx).Any("cohort", cohort).Msg("success to get user cohort")
}
