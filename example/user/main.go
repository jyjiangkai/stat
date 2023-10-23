package main

import (
	"context"
	"fmt"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/mailchimp"
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
	mailchimpCfg := mailchimp.Config{
		Enable:     true,
		WebhookUrl: "https://0lj80uzusozxlooq.connector.vanus.ai/api/v1/source/http/650439c93f0b52737fe5b8d0",
	}
	mailchimp.Init(ctx, mailchimpCfg)

	svc := services.NewUserService(cli)
	err = svc.Start()
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to start user service")
		return
	}
	log.Info(ctx).Msg("success to start user service")
}
