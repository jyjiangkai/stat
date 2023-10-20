package main

import (
	"context"
	"fmt"

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

	svc := services.NewActionService(cli)
	// pg := api.Page{
	// 	PageSize: 2,
	// }
	// filter := api.Filter{}
	// opts := &api.ListOptions{
	// 	KindSelector: "connect",
	// 	TypeSelector: "marketing",
	// }
	err = svc.Start()
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to start action service")
		return
	}
	log.Info(ctx).Msg("success to start action service")
}
