package services

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jyjiangkai/stat/api"
	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
)

const (
	// Prod         = "vanus-cloud-prod"
	billCollName = "bills"
)

type CollectorService struct {
	mgoCli              *mongo.Client
	connectionColl      *mongo.Collection
	userColl            *mongo.Collection
	quotaColl           *mongo.Collection
	billColl            *mongo.Collection
	aiBillColl          *mongo.Collection
	aiAppColl           *mongo.Collection
	aiUploadColl        *mongo.Collection
	aiKnowledgeBaseColl *mongo.Collection
	statisticColl       *mongo.Collection
	closeC              chan struct{}
}

func NewCollectorService(cli *mongo.Client) *CollectorService {
	return &CollectorService{
		mgoCli:              cli,
		connectionColl:      cli.Database(db.GetDatabaseName()).Collection("connections"),
		userColl:            cli.Database(db.GetDatabaseName()).Collection("users"),
		quotaColl:           cli.Database(db.GetDatabaseName()).Collection("quotas"),
		billColl:            cli.Database(db.GetDatabaseName()).Collection("bills"),
		aiBillColl:          cli.Database(db.GetDatabaseName()).Collection("ai_bills"),
		aiAppColl:           cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		aiUploadColl:        cli.Database(db.GetDatabaseName()).Collection("ai_upload"),
		aiKnowledgeBaseColl: cli.Database(db.GetDatabaseName()).Collection("ai_knowledge_bases"),
		statisticColl:       cli.Database(db.GetDatabaseName()).Collection("statistics"),
		closeC:              make(chan struct{}),
	}
}

func (cs *CollectorService) Start() error {
	ctx := context.Background()

	now := time.Now()
	log.Info(ctx).Msgf("start collect at: %+v\n", now)
	err := cs.Collect(ctx, "full")
	if err != nil {
		log.Error(ctx).Err(err).Msgf("Collect user stat failed at %+v\n", now)
	} else {
		log.Info(ctx).Msgf("Collect user stat success at %+v\n", now)
	}

	// go func() {
	// 	ticker := time.NewTicker(time.Hour)
	// 	defer ticker.Stop()
	// 	defer log.Warn(ctx).Err(nil).Msg("collect routine exit")
	// 	for {
	// 		select {
	// 		case <-cs.closeC:
	// 			log.Info(ctx).Msg("Bill Service stopped.")
	// 			return
	// 		case <-ticker.C:
	// 			now := time.Now()
	// 			if now.Hour() == 2 {
	// 				log.Info(ctx).Msgf("start collect at: %+v\n", now)
	// 				err := cs.Collect(ctx, "")
	// 				if err != nil {
	// 					log.Warn(ctx).Msgf("Collect bill failed at %+v\n", now)
	// 				} else {
	// 					log.Info(ctx).Msgf("Collect bill success at %+v\n", now)
	// 				}
	// 			}
	// 		}
	// 	}
	// }()
	return nil
}

func (cs *CollectorService) Stop() error {
	return nil
}

func (cs *CollectorService) Collect(ctx context.Context, kind string) error {
	if kind == "" {
		log.Error(ctx).Msgf("collect kind is null")
		return api.ErrInvalidParameter
	}

	var err error
	switch kind {
	case "full":
		err = cs.fullCollect(ctx)
		if err != nil {
			return err
		}
	case "incremental":
		err = cs.incrementalCollect(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (cs *CollectorService) fullCollect(ctx context.Context) error {
	var (
		skip int64 = 0
		// limit int64  = 1
		sort bson.M = bson.M{"created_at": 1}
	)

	start := time.Date(2023, 8, 31, 0, 0, 0, 0, time.UTC)
	end := time.Date(2023, 9, 1, 0, 0, 0, 0, time.UTC)
	query := bson.M{
		"created_at": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}
	// query := bson.M{
	// 	"oidc_id": "google-oauth2|113887992409437297567",
	// }
	opt := options.FindOptions{
		// Limit: &limit,
		Skip: &skip,
		Sort: sort,
	}
	cursor, err := cs.userColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("failed to find user cause no documents")
			return nil
		}
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	cnt := 0
	for cursor.Next(ctx) {
		start := time.Now()
		user := &cloud.User{}
		if err = cursor.Decode(user); err != nil {
			return err
		}
		cnt += 1
		bills, err := cs.getBills(ctx, user.OID)
		if err != nil {
			return err
		}
		class, err := cs.getClass(ctx, user.OID)
		if err != nil {
			return err
		}
		usage, err := cs.getUsages(ctx, user.OID)
		if err != nil {
			return err
		}
		statUser := &models.User{
			Base:         user.Base,
			OID:          user.OID,
			Phone:        user.Phone,
			Email:        user.Email,
			Country:      user.Country,
			FamilyName:   user.FamilyName,
			GivenName:    user.GivenName,
			NickName:     user.NickName,
			CompanyName:  user.CompanyName,
			CompanyEmail: user.CompanyEmail,
			Class:        class,
			Bills:        bills,
			Usages:       usage,
		}
		_, err = cs.statisticColl.InsertOne(ctx, statUser)
		if err != nil {
			return db.HandleDBError(err)
		}
		log.Info(ctx).Msgf("[%d] spent %d ms to statistic user %s\n", cnt, time.Since(start).Milliseconds(), user.OID)
	}
	fmt.Printf("success to stat users, cnt: %d\n", cnt)
	return nil
}

func (cs *CollectorService) incrementalCollect(ctx context.Context) error {
	return nil
}
