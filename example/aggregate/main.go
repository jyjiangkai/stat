package main

import (
	"context"
	"fmt"
	"time"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models/cloud"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	// billColl := cli.Database(db.GetDatabaseName()).Collection("ai_bills")
	// actionColl := cli.Database("vanus_user_analytics").Collection("user_actions")
	dailyStatsColl := cli.Database("vanus-user-statistics").Collection("daily_stats")

	// FindOne(ctx, billColl)
	// CountDocuments(ctx, actionColl)
	DeleteMany(ctx, dailyStatsColl)
	// Aggregate(ctx, billColl)

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

func CountDocuments(ctx context.Context, coll *mongo.Collection) error {
	query := bson.M{
		"website": bson.M{
			"$regex": "https://ai.vanustest.com",
		},
	}
	cnt, err := coll.CountDocuments(ctx, query)
	if err != nil {
		return err
	}
	log.Info(ctx).Int64("cnt", cnt).Msg("success to count documents")
	return nil
}

func DeleteMany(ctx context.Context, coll *mongo.Collection) error {
	query := bson.M{}
	_, err := coll.DeleteMany(ctx, query)
	if err != nil {
		return err
	}
	log.Info(ctx).Msg("success to delete many")
	return nil
}

func Aggregate(ctx context.Context, coll *mongo.Collection) error {
	// ct := time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)
	id, _ := primitive.ObjectIDFromHex("64c7181cc7f602af76f1ca21")
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"app_id", id},
			}},
		},
		// {
		// 	{"$project", bson.D{
		// 		{"adjustedDate", bson.D{
		// 			{"$subtract", []interface{}{"$collected_at", 86400000}},
		// 			{"usage_num", 1},
		// 		}},
		// 	}},
		// },
		// {
		// 	{"$project", bson.D{
		// 		{"date", bson.D{
		// 			{"$dateToString", bson.M{"format": "%Y-%m-%d", "date": "$adjustedDate"}},
		// 		}},
		// 		{"usage_num", 1},
		// 	}},
		// },
		{
			{"$project", bson.D{
				{"date", bson.D{
					{"$dateToString", bson.M{"format": "%Y-%m-%d", "date": "$collected_at"}},
				}},
				{"chatgpt_3_5", "$usage.chatgpt_3_5"},
				{"chatgpt_4", "$usage.chatgpt_4"},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$date"},
				{"totalChatGPT_3_5", bson.M{"$sum": "$chatgpt_3_5"}},
				{"totalChatGPT_4", bson.M{"$sum": "$chatgpt_4"}},
			}},
		},
		// {
		// 	{"$group", bson.D{
		// 		{"_id", "$date"},
		// 		{"usage", bson.D{
		// 			{"$sum", bson.D{
		// 				{"$add", []interface{}{
		// 					"$usage.chatgpt_3_5",
		// 					bson.M{"$multiply": []interface{}{"$usage.chatgpt_4", 20}},
		// 				}},
		// 			}},
		// 		}},
		// 	}},
		// },
		{
			{"$sort", bson.D{
				{"_id", -1},
			}},
		},
	}
	type usageGroup struct {
		Date      string `bson:"_id"`
		ChatGPT35 uint64 `bson:"totalChatGPT_3_5"`
		ChatGPT4  uint64 `bson:"totalChatGPT_4"`
		Usage     uint64 `bson:"usage"`
	}
	cursor, err := coll.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
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
