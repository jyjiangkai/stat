package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/monitor"
)

const (
	UsageAlarmThreshold = 30 // trigger an alarm when usage decreases by 30% or more
)

type AlarmService struct {
	mgoCli         *mongo.Client
	connectionColl *mongo.Collection
	userColl       *mongo.Collection
	quotaColl      *mongo.Collection
	billColl       *mongo.Collection
	aiBillColl     *mongo.Collection
	closeC         chan struct{}
}

func NewAlarmService(cli *mongo.Client) *AlarmService {
	return &AlarmService{
		mgoCli:         cli,
		connectionColl: cli.Database(db.GetDatabaseName()).Collection("connections"),
		userColl:       cli.Database(db.GetDatabaseName()).Collection("users"),
		quotaColl:      cli.Database(db.GetDatabaseName()).Collection("quotas"),
		billColl:       cli.Database(db.GetDatabaseName()).Collection("bills"),
		aiBillColl:     cli.Database(db.GetDatabaseName()).Collection("ai_bills"),
		closeC:         make(chan struct{}),
	}
}

func (as *AlarmService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("alarm routine exit")
		for {
			select {
			case <-as.closeC:
				log.Info(ctx).Msg("alarm service stopped.")
				return
			case <-ticker.C:
				now := time.Now()
				if now.Hour() == 2 {
					log.Info(ctx).Msgf("start alarm of usage at: %+v\n", now)
					err := as.AlarmOfUsage(ctx, now)
					if err != nil {
						log.Error(ctx).Err(err).Msgf("usage alarm failed at %+v\n", time.Now())
					}
				}
			}
		}
	}()
	return nil
}

func (as *AlarmService) Stop() error {
	return nil
}

func (as *AlarmService) AlarmOfUsage(ctx context.Context, now time.Time) error {
	err := as.AlarmOfConnectUsage(ctx, now)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to alarm of connect usage")
	}
	err = as.AlarmOfAIUsage(ctx, now)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to alarm of ai usage")
	}
	return nil
}

func (as *AlarmService) AlarmOfConnectUsage(ctx context.Context, now time.Time) error {
	latest := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	before := latest.Add(-24 * time.Hour)
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"collected_at", bson.M{"$in": []interface{}{latest, before}}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$collected_at"},
				{"usage", bson.D{
					{"$sum", "$delivered_num"},
				}},
			}},
		},
		{
			{"$sort", bson.D{
				{"_id", -1},
			}},
		},
	}
	type usageGroup struct {
		Date  time.Time `bson:"_id"`
		Usage uint64    `bson:"usage"`
	}
	cursor, err := as.billColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return err
	}
	defer cursor.Close(ctx)
	usage := make([]usageGroup, 0)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return err
		}
		usage = append(usage, usageGroup)
	}
	if len(usage) != 2 {
		log.Error(ctx).Msgf("usage group len is %d\n", len(usage))
		return err
	}
	if usage[0].Usage >= usage[1].Usage {
		log.Info(ctx).Msg("increased usage, no need for alarm")
		return nil
	}
	decrease, needAlarm := ExceedingTheUsageAlarmThreshold(usage[0].Usage, usage[1].Usage)
	if needAlarm {
		err = monitor.SendAlarm(ctx, fmt.Sprintf("On %s %dth, the usage of connect was %d, decrease of %0.2f%% compared to the previous day", now.Month().String(), now.Day(), usage[0].Usage, decrease))
		if err != nil {
			return err
		}
	}
	return nil
}

func (as *AlarmService) AlarmOfAIUsage(ctx context.Context, now time.Time) error {
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	latestUsage, err := as.getAIDailyUsage(ctx, date)
	if err != nil {
		return err
	}
	beforeUsage, err := as.getAIDailyUsage(ctx, date.Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if latestUsage >= beforeUsage {
		log.Info(ctx).Msg("increased usage, no need alarm")
		return nil
	}
	decrease, needAlarm := ExceedingTheUsageAlarmThreshold(latestUsage, beforeUsage)
	if needAlarm {
		err = monitor.SendAlarm(ctx, fmt.Sprintf("On %s %dth, the usage of ai was %d, decrease of %0.2f%% compared to the previous day", now.Month().String(), now.Day(), latestUsage, decrease))
		if err != nil {
			return err
		}
	} else {
		log.Info(ctx).Msg("decrease usage has not reached the alarm threshold, no need alarm")
	}
	return nil
}

func (as *AlarmService) getAIDailyUsage(ctx context.Context, date time.Time) (uint64, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{
				"collected_at": bson.M{
					"$gt":  date.Add(-24 * time.Hour),
					"$lte": date,
				},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", bson.M{
					"$dateToString": bson.M{
						"format": date.Format("2006-01-02T"),
						"date":   "$collected_at",
					},
				}},
				{"usage", bson.D{
					{"$sum", bson.D{
						{"$add", []interface{}{
							"$usage.chatgpt_3_5",
							bson.M{"$multiply": []interface{}{"$usage.chatgpt_4", 20}},
						}},
					}},
				}},
			}},
		},
	}
	type usageGroup struct {
		Date  string `bson:"_id"`
		Usage uint64 `bson:"usage"`
	}
	cursor, err := as.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return 0, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return 0, err
		}
		return usageGroup.Usage, nil
	}
	return 0, errors.New("no usage group")
}

func ExceedingTheUsageAlarmThreshold(latest, before uint64) (float64, bool) {
	new := float64(latest)
	old := float64(before)
	decrease := (old - new) / old * 100
	if decrease > UsageAlarmThreshold {
		return decrease, true
	}
	return 0, false
}
