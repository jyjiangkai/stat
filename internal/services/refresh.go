package services

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"github.com/jyjiangkai/stat/utils"
)

const (
	BatchSize = 2000
)

type RefreshService struct {
	mgoCli              *mongo.Client
	connectionColl      *mongo.Collection
	userColl            *mongo.Collection
	quotaColl           *mongo.Collection
	billColl            *mongo.Collection
	aiBillColl          *mongo.Collection
	aiAppColl           *mongo.Collection
	aiUploadColl        *mongo.Collection
	aiKnowledgeBaseColl *mongo.Collection
	statColl            *mongo.Collection
	wg                  sync.WaitGroup
	closeC              chan struct{}
}

func NewRefreshService(cli *mongo.Client) *RefreshService {
	return &RefreshService{
		mgoCli:              cli,
		connectionColl:      cli.Database(db.GetDatabaseName()).Collection("connections"),
		userColl:            cli.Database(db.GetDatabaseName()).Collection("users"),
		quotaColl:           cli.Database(db.GetDatabaseName()).Collection("quotas"),
		billColl:            cli.Database(db.GetDatabaseName()).Collection("bills"),
		aiBillColl:          cli.Database(db.GetDatabaseName()).Collection("ai_bills"),
		aiAppColl:           cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		aiUploadColl:        cli.Database(db.GetDatabaseName()).Collection("ai_upload"),
		aiKnowledgeBaseColl: cli.Database(db.GetDatabaseName()).Collection("ai_knowledge_bases"),
		statColl:            cli.Database(db.GetDatabaseName()).Collection("stats"),
		closeC:              make(chan struct{}),
	}
}

func (rs *RefreshService) Start() error {
	ctx := context.Background()
	go func() {
		now := time.Now()
		log.Info(ctx).Msgf("start refresh user stat at: %+v\n", now)
		err := rs.Refresh(ctx, now)
		if err != nil {
			log.Error(ctx).Err(err).Msgf("refresh user stat failed at %+v\n", time.Now())
		}
		log.Info(ctx).Msgf("finish refresh user stat at: %+v\n", time.Now())
	}()
	// go func() {
	// 	ticker := time.NewTicker(time.Hour)
	// 	defer ticker.Stop()
	// 	defer log.Warn(ctx).Err(nil).Msg("refresh routine exit")
	// 	for {
	// 		select {
	// 		case <-rs.closeC:
	// 			log.Info(ctx).Msg("refresh service stopped.")
	// 			return
	// 		case <-ticker.C:
	// 			now := time.Now()
	// 			if now.Hour() == 1 {
	// 				log.Info(ctx).Msgf("start refresh user stat at: %+v\n", now)
	// 				err := rs.Refresh(ctx, now)
	// 				if err != nil {
	// 					log.Error(ctx).Err(err).Msgf("refresh user stat failed at %+v\n", time.Now())
	// 				}
	// 				log.Info(ctx).Msgf("finish refresh user stat at: %+v\n", time.Now())
	// 			}
	// 		}
	// 	}
	// }()
	return nil
}

func (rs *RefreshService) Stop() error {
	return nil
}

func (rs *RefreshService) Refresh(ctx context.Context, now time.Time) error {
	query := bson.M{}
	cnt, err := rs.userColl.CountDocuments(ctx, query)
	if err != nil {
		return err
	}
	log.Info(ctx).Msgf("current collection time is %+v, with a total of %d users\n", now, cnt)
	step := int64(BatchSize)
	goroutines := 0
	for i := int64(0); i < cnt; {
		start := i
		end := i + step
		if end > cnt {
			end = cnt
		}
		rs.wg.Add(1)
		goroutines += 1
		go rs.rangeRefresh(ctx, start, end, now)
		i += step
	}
	log.Info(ctx).Msgf("launch a total of %d goroutines, with each goroutine assigned %d user collection tasks\n", goroutines, step)
	log.Info(ctx).Msg("starting collection, please wait...")
	rs.wg.Wait()
	log.Info(ctx).Msgf("finished user data collection, spent %f seconds, updated %d users\n", time.Since(now).Seconds(), cnt)
	return nil
}

func (rs *RefreshService) rangeRefresh(ctx context.Context, start int64, end int64, now time.Time) {
	var (
		reterr error
		cnt    int    = 0
		skip   int64  = start
		limit  int64  = end - start
		sort   bson.M = bson.M{"created_at": 1}
	)
	// log.Info(ctx).Msgf("start goroutine for range refresh, start: %d, end: %d\n", start, end)
	defer rs.wg.Done()
	defer func() {
		if reterr != nil {
			log.Error(ctx).Err(reterr).Int64("start", start).Int64("end", end).Msg("failed to refresh users")
		}
		log.Info(ctx).Msgf("finish goroutine for range[%d, %d] refresh, %d completed, %d remaining.\n", start, end, cnt, (limit - int64(cnt)))
	}()
	query := bson.M{}
	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := rs.userColl.Find(ctx, query, &opt)
	if err != nil {
		reterr = err
		log.Error(ctx).Err(err).Msg("failed to find user")
		return
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	for cursor.Next(ctx) {
		user := &cloud.User{}
		if err = cursor.Decode(user); err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to decode user")
			return
		}
		cnt += 1
		bills, err := rs.getBills(ctx, user.OID, now)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to get bills")
			return
		}
		class, err := rs.getClass(ctx, user.OID, now)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to get class")
			return
		}
		usage, err := rs.getUsages(ctx, user.OID)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to get usages")
			return
		}
		cohort, err := rs.GetCohort(ctx, user)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to get cohort")
			return
		}
		user.Base.UpdatedAt = now
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
			Industry:     rs.GetUserIndustry(ctx, user),
			Class:        class,
			Bills:        bills,
			Usages:       usage,
			Cohort:       cohort,
		}
		query := bson.M{
			"_id": user.ID,
		}
		opts := &options.ReplaceOptions{
			Upsert: utils.PtrBool(true),
		}
		_, err = rs.statColl.ReplaceOne(ctx, query, statUser, opts)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to insert stat user")
			return
		}
		// log.Info(ctx).Msgf("[%d] spent %d ms to refresh user stat: %s\n", cnt, time.Since(start).Milliseconds(), user.OID)
	}
}

func (rs *RefreshService) getLastRefreshTime(ctx context.Context) (time.Time, error) {
	var (
		sortBy string    = "updated_at"
		now    time.Time = time.Now()
	)
	query := bson.M{}
	opt := options.FindOneOptions{
		Sort: bson.M{sortBy: -1},
	}
	result := rs.statColl.FindOne(ctx, query, &opt)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			log.Error(ctx).Err(result.Err()).Msg("get last refresh stat but no document")
		}
		return now, result.Err()
	}
	user := &models.User{}
	if err := result.Decode(user); err != nil {
		return now, db.HandleDBError(err)
	}
	return user.UpdatedAt, nil
}

func (rs *RefreshService) GetRegisteredUserList(ctx context.Context, start, end time.Time) ([]string, error) {
	users := make([]string, 0)
	query := bson.M{
		"created_at": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}
	cursor, err := rs.userColl.Find(ctx, query)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find user but no documents")
			return users, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	for cursor.Next(ctx) {
		user := &cloud.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, err
		}
		users = append(users, user.OID)
	}
	return users, nil
}

func (rs *RefreshService) GetAIBillChangedUserList(ctx context.Context, start, end time.Time) ([]string, error) {
	users := make([]string, 0)
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"collected_at", bson.D{
					{"$gte", start},
					{"$lte", end},
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$user_id"},
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
		UserID string `bson:"_id"`
		Usage  uint64 `bson:"usage"`
	}
	cursor, err := rs.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find ai bills but no documents")
			return users, nil
		}
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return nil, err
		}
		users = append(users, usageGroup.UserID)
	}
	return users, nil
}

func (rs *RefreshService) GetConnectBillChangedUserList(ctx context.Context, start, end time.Time) ([]string, error) {
	users := make([]string, 0)
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"collected_at", bson.D{
					{"$gte", start},
					{"$lte", end},
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$user_id"},
				{"usage", bson.D{
					{"$sum", "$usage_num"},
				}},
			}},
		},
	}
	type usageGroup struct {
		UserID string `bson:"_id"`
		Usage  uint64 `bson:"usage"`
	}
	cursor, err := rs.billColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find connect bills but no documents")
			return users, nil
		}
		return nil, err
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return nil, err
		}
		users = append(users, usageGroup.UserID)
	}
	return users, nil
}

func (rs *RefreshService) GetUserIndustry(ctx context.Context, user *cloud.User) string {
	if user.Industry == "Others" {
		return user.IndustryExtra
	}
	return user.Industry
}
