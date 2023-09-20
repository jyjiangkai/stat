package services

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
)

var (
	StartAt time.Time = time.Date(2023, 3, 9, 0, 0, 0, 0, time.UTC)
)

type CohortService struct {
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
	cohortColl          *mongo.Collection
	wg                  sync.WaitGroup
	closeC              chan struct{}
}

func NewCohortService(cli *mongo.Client) *CohortService {
	return &CohortService{
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
		cohortColl:          cli.Database(db.GetDatabaseName()).Collection("cohorts"),
		closeC:              make(chan struct{}),
	}
}

func (cs *CohortService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("cohort analysis routine exit")
		for {
			select {
			case <-cs.closeC:
				log.Info(ctx).Msg("cohort analysis service stopped.")
				return
			case <-ticker.C:
				now := time.Now()
				if now.Weekday() == time.Monday && now.Hour() == 2 {
					log.Info(ctx).Msgf("start cohort analysis at: %+v\n", now)
					err := cs.CohortAnalysis(ctx, now)
					if err != nil {
						log.Warn(ctx).Msgf("cohort analysis failed at %+v\n", now)
					}
				}
			}
		}
	}()
	return nil
}

func (cs *CohortService) Stop() error {
	return nil
}

func (cs *CohortService) CohortAnalysis(ctx context.Context, now time.Time) error {
	// query := bson.M{}
	// cnt, err := rs.userColl.CountDocuments(ctx, query)
	// if err != nil {
	// 	return err
	// }
	// log.Info(ctx).Msgf("current collection time is %+v, with a total of %d users\n", now, cnt)
	// week := GetFirstWeek(StartAt)
	// for true {
	// 	// 1. 通过该week查询mongo，判断是否已插入cohort
	// 	_, exist, err := cs.GetCohortFromWeek(ctx, week)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	// 2. 如果已有cohort，则获取该cohort的lastWeek的nextWeek，获取nextWeek的数据并增量更新到该cohort上
	// 	if exist {
	// 		// TODO(jiangkai): 增量更新
	// 		continue
	// 	}
	// 	// 3. 如果没有cohort，获取当前week内的所有用户，并计算这一组用户在每一周的留存率，最后插入一条新的cohort数据
	// 	_, err = cs.GetUsersFromWeek(ctx, week)
	// 	if err != nil {
	// 		return err
	// 	}

	// 	// 4. 判断该week是否已经是最新的week，如果是，则退出循环
	// 	// 5. 如果该week不是最新的week，则将week重置为nextWeek
	// }
	return nil
}

func (cas *CohortService) GetRetentions(ctx context.Context, week *models.Week, users []*models.User) ([]*models.Retention, error) {
	query := bson.M{}
	cnt, err := cas.statColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	log.Info(ctx).Msgf("current collection time is %+v, with a total of %d users\n", time.Now(), cnt)
	step := int64(500)
	goroutines := 0
	for i := int64(0); i < cnt; {
		// start := i
		end := i + step
		if end > cnt {
			end = cnt
		}
		cas.wg.Add(1)
		goroutines += 1
		// go cas.rangeRefresh(ctx, start, end, now)
		i += step
	}
	log.Info(ctx).Msgf("launch a total of %d goroutines, with each goroutine assigned %d user collection tasks\n", goroutines, step)
	log.Info(ctx).Msg("starting collection, please wait...")
	cas.wg.Wait()
	return nil, nil
}

func (cas *CohortService) GetUsersFromWeek(ctx context.Context, week *models.Week) ([]*models.User, error) {
	users := make([]*models.User, 0)
	query := bson.M{
		"created_at": bson.M{
			"$gte": week.Start,
			"$lte": week.End,
		},
	}
	cursor, err := cas.statColl.Find(ctx, query)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find stat but no documents")
			return users, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}
func (cs *CohortService) GetCohortFromWeek(ctx context.Context, week *models.Week) (*models.Cohort, bool, error) {
	query := bson.M{
		"week.number": week.Number,
	}
	result := cs.cohortColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, false, nil
		}
	}
	cohort := &models.Cohort{}
	if err := result.Decode(cohort); err != nil {
		log.Error(ctx).Err(err).Msg("failed to decode user quota")
		return nil, false, db.HandleDBError(err)
	}
	return cohort, true, nil
}

func TimeToWeek(at time.Time) *models.Week {
	// 获取当前时间所在的周一和周日的时间
	weekday := at.Weekday()
	startOffset := time.Duration((weekday + 6) % 7)
	endOffset := time.Duration((7 - weekday) % 7)

	start := at.Add(-startOffset * 24 * time.Hour).Truncate(24 * time.Hour)
	end := at.Add(endOffset * 24 * time.Hour).Truncate(24 * time.Hour)

	return &models.Week{
		Number: 0,
		Alias:  fmt.Sprintf("%s %d, %d", start.Month().String(), start.Day(), start.Year()),
		Start:  start,
		End:    end,
	}
}

func GetNextWeek(week *models.Week) *models.Week {
	return &models.Week{
		Number: week.Number + 1,
		Alias:  fmt.Sprintf("week %02d", week.Number+1),
		Start:  week.End,
		End:    week.End.Add(7 * 24 * time.Hour),
	}
}

func (rs *RefreshService) GetCohort(ctx context.Context, user *cloud.User) (*models.Cohort, error) {
	week := TimeToWeek(user.CreatedAt)
	aiRetention, err := rs.getAIRetention(ctx, user.OID, week)
	if err != nil {
		return nil, err
	}
	ctRetention, err := rs.getConnectRetention(ctx, user.OID, week)
	if err != nil {
		return nil, err
	}
	return &models.Cohort{
		Week:    week,
		AI:      aiRetention,
		Connect: ctRetention,
	}, nil
}

func (rs *RefreshService) getAIRetention(ctx context.Context, oid string, week0 *models.Week) (map[string]*models.Retention, error) {
	results := make(map[string]*models.Retention)
	week := &models.Week{
		Number: week0.Number,
		Alias:  fmt.Sprintf("week %02d", week0.Number),
		Start:  week0.Start,
		End:    week0.End,
	}
	for true {
		retention, err := rs.getAIWeekUsage(ctx, oid, week)
		if err != nil {
			return nil, err
		}
		results[week.Alias] = retention
		week = GetNextWeek(week)
		if week.End.After(time.Now()) {
			break
		}
	}
	return results, nil
}

func (rs *RefreshService) getAIWeekUsage(ctx context.Context, oid string, week *models.Week) (*models.Retention, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{
				"user_id": oid,
				"collected_at": bson.M{
					"$gt":  week.Start,
					"$lte": week.End,
				},
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
		User  string `bson:"_id"`
		Usage uint64 `bson:"usage"`
	}
	cursor, err := rs.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	usage := make([]usageGroup, 0)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return nil, err
		}
		// log.Info(ctx).Any("usage_group", usageGroup).Msg("success to get ai usage group")
		usage = append(usage, usageGroup)
	}
	if len(usage) == 0 {
		return &models.Retention{
			Week:  week,
			Usage: 0,
		}, nil
	}
	if len(usage) != 1 {
		return nil, errors.New("get ai week usage failed, usage len error")
	}
	retention := &models.Retention{
		Week:  week,
		Usage: usage[0].Usage,
	}
	if retention.Usage != 0 {
		retention.Active = true
	}
	return retention, nil
}

func (rs *RefreshService) getConnectRetention(ctx context.Context, oid string, week0 *models.Week) (map[string]*models.Retention, error) {
	results := make(map[string]*models.Retention)
	week := &models.Week{
		Number: week0.Number,
		Alias:  fmt.Sprintf("week %02d", week0.Number),
		Start:  week0.Start,
		End:    week0.End,
	}
	for {
		retention, err := rs.getConnectWeekUsage(ctx, oid, week)
		if err != nil {
			return nil, err
		}
		results[week.Alias] = retention
		week = GetNextWeek(week)
		if week.End.After(time.Now()) {
			break
		}
	}
	return results, nil
}

func (rs *RefreshService) getConnectWeekUsage(ctx context.Context, oid string, week *models.Week) (*models.Retention, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{
				"user_id": oid,
				"collected_at": bson.M{
					"$gt":  week.Start,
					"$lte": week.End,
				},
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
		User  string `bson:"_id"`
		Usage uint64 `bson:"usage"`
	}
	cursor, err := rs.billColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	usage := make([]usageGroup, 0)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return nil, err
		}
		// log.Info(ctx).Any("usage_group", usageGroup).Msg("success to get ct usage group")
		usage = append(usage, usageGroup)
	}
	if len(usage) == 0 {
		return &models.Retention{
			Week:  week,
			Usage: 0,
		}, nil
	}
	if len(usage) != 1 {
		return nil, errors.New("get connect week usage failed, usage len error")
	}
	retention := &models.Retention{
		Week:  week,
		Usage: usage[0].Usage,
	}
	if retention.Usage != 0 {
		retention.Active = true
	}
	return retention, nil
}
