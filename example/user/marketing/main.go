package main

import (
	"context"
	"fmt"

	"github.com/jyjiangkai/stat/api"
	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/mailchimp"
	"github.com/jyjiangkai/stat/models"
)

var (
	Database         string = "vanus-cloud-prod"
	Username         string = "vanus-cloud-prod-rw"
	Password         string = ""
	Address          string = "cluster1.odfrc.mongodb.net"
	MailChimpWebhook string = "https://0lj80uzusozxlooq.connector.vanus.ai/api/v1/source/http/650439c93f0b52737fe5b8d0"
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

	mailchimp.Init(ctx, mailchimp.Config{Enable: true, WebhookUrl: MailChimpWebhook})

	svc := services.NewUserService(cli)
	pg := api.Page{}
	filter := api.Filter{}
	opts := &api.ListOptions{
		KindSelector: "ai",
		TypeSelector: "marketing",
	}
	result, err := svc.List(ctx, pg, filter, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to list weekly marketing user")
		return
	}
	cnt := 0
	for idx := range result.List {
		user := result.List[idx].(*models.User)
		if mailchimp.ValidateEmail(user.Email) {
			err := mailchimp.AddMember(ctx, user.Email)
			if err != nil {
				log.Error(ctx).Str("email", user.Email).Msg("failed to add member to mailchimp")
			}
			log.Info(ctx).Str("email", user.Email).Msg("success to add member to mailchimp")
			cnt += 1
		}
	}
	log.Info(ctx).Int("cnt", cnt).Msg("success to list weekly marketing user and upload to mailchimp")
}
