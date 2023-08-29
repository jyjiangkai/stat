package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models/cloud"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
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

	billColl := cli.Database(db.GetDatabaseName()).Collection("bills")

	// FindOne(ctx, billColl)
	Aggregate(ctx, billColl)

	return
}

func FindOne(ctx context.Context, coll *mongo.Collection) error {
	now := time.Now()
	query := bson.M{
		// "created_by": oid,
		// "plan.kind":  kind,
		"collected_at": bson.M{
			"$lte": now,
		},
		// "period_of_validity.end": bson.M{
		// 	"$gte": now,
		// },
	}

	result := coll.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		return result.Err()
	}
	bill := &cloud.Bill{}
	if err := result.Decode(bill); err != nil {
		log.Error(ctx).Err(err).Msg("failed to decode user quota")
		return err
	}
	log.Info(ctx).Msgf("success get bill: %+v\n", bill)
	return nil
}

func Aggregate(ctx context.Context, coll *mongo.Collection) error {
	ct := time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				// {"user_id", utils.GetUserID(ctx)},
				{"collected_at", bson.D{
					{"$gte", ct},
					//{"$lte", endTime},
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$user_id"},
				{"usage_num", bson.D{
					{"$sum", "$usage_num"},
				}},
			}},
		},
	}
	type usageGroup struct {
		UserID   string `bson:"_id"`
		UsageNum uint64 `bson:"usage_num"`
	}
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		return err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return err
		}
		log.Info(ctx).Msgf("success get usage group: %+v\n", usageGroup)
	}
	return nil
}
