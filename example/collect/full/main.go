package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/internal/services"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var (
	Database string = "vanus-cloud-prod"
	Username string = "vanus-cloud-prod-rw"
	Password string = ""
	Address  string = "cluster1.odfrc.mongodb.net"
)

func Init(ctx context.Context) (*mongo.Client, error) {
	var (
		err error
	)
	uri := fmt.Sprintf("mongodb+srv://%s:%s@%s/?retryWrites=true&w=majority", Username, Password, Address)
	clientOptions := options.Client().
		ApplyURI(uri).
		SetServerAPIOptions(options.ServerAPI(options.ServerAPIVersion1))
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cli, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, err
	}

	if err = cli.Ping(ctx, readpref.Primary()); err != nil {
		return nil, err
	}
	fmt.Printf("Success to connect to MongoDB, address: %s\n", Address)

	if cli != nil {
		return cli, nil
	}

	return cli, nil
}

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

	collector := services.NewCollectorService(cli)
	collector.Start()
}
