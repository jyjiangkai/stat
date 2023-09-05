package services

import (
	"context"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
)

var (
	StartAt time.Time = time.Date(2023, 3, 9, 0, 0, 0, 0, time.UTC)
)

type CohortAnalysisService struct {
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

func NewCohortAnalysisService(cli *mongo.Client) *CohortAnalysisService {
	return &CohortAnalysisService{
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

func (rs *CohortAnalysisService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("cohort analysis routine exit")
		for {
			select {
			case <-rs.closeC:
				log.Info(ctx).Msg("cohort analysis service stopped.")
				return
			case <-ticker.C:
				now := time.Now()
				if now.Weekday() == time.Monday && now.Hour() == 2 {
					log.Info(ctx).Msgf("start cohort analysis at: %+v\n", now)
					err := rs.CohortAnalysis(ctx, now)
					if err != nil {
						log.Warn(ctx).Msgf("cohort analysis failed at %+v\n", now)
					}
				}
			}
		}
	}()
	return nil
}

func (rs *CohortAnalysisService) Stop() error {
	return nil
}

func (cas *CohortAnalysisService) CohortAnalysis(ctx context.Context, now time.Time) error {
	// query := bson.M{}
	// cnt, err := rs.userColl.CountDocuments(ctx, query)
	// if err != nil {
	// 	return err
	// }
	// log.Info(ctx).Msgf("current collection time is %+v, with a total of %d users\n", now, cnt)
	week := GetFirstWeek(StartAt)
	for true {
		// 1. 通过该week查询mongo，判断是否已插入cohort
		_, exist, err := cas.GetCohortFromWeek(ctx, week)
		if err != nil {
			return err
		}
		// 2. 如果已有cohort，则获取该cohort的lastWeek的nextWeek，获取nextWeek的数据并增量更新到该cohort上
		if exist {
			// TODO(jiangkai): 增量更新
			continue
		}
		// 3. 如果没有cohort，获取当前week内的所有用户，并计算这一组用户在每一周的留存率，最后插入一条新的cohort数据
		_, err = cas.GetUsersFromWeek(ctx, week)
		if err != nil {
			return err
		}

		// 4. 判断该week是否已经是最新的week，如果是，则退出循环
		// 5. 如果该week不是最新的week，则将week重置为nextWeek
	}
	return nil
}

func (cas *CohortAnalysisService) GetRetentions(ctx context.Context, week *models.Week, users []*models.User) ([]*models.Retention, error) {
	query := bson.M{}
	cnt, err := cas.statColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	log.Info(ctx).Msgf("current collection time is %+v, with a total of %d users\n", time.Now(), cnt)
	step := int64(500)
	goroutines := 0
	for i := int64(0); i < cnt; {
		start := i
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

func (cas *CohortAnalysisService) GetUsersFromWeek(ctx context.Context, week *models.Week) ([]*models.User, error) {
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
func (cas *CohortAnalysisService) GetCohortFromWeek(ctx context.Context, week *models.Week) (*models.CohortAnalysis, bool, error) {
	query := bson.M{
		"week.number": week.Number,
	}
	result := cas.cohortColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, false, nil
		}
	}
	cohort := &models.CohortAnalysis{}
	if err := result.Decode(cohort); err != nil {
		log.Error(ctx).Err(err).Msg("failed to decode user quota")
		return nil, false, db.HandleDBError(err)
	}
	return cohort, true, nil
}

func GetFirstWeek(startAt time.Time) *models.Week {
	// 获取当前时间所在的周一和周日的时间
	weekday := startAt.Weekday()
	startOffset := time.Duration((weekday + 6) % 7)
	endOffset := time.Duration((7 - weekday) % 7)

	start := startAt.Add(-startOffset * 24 * time.Hour).Truncate(24 * time.Hour)
	end := startAt.Add(endOffset * 24 * time.Hour).Truncate(24 * time.Hour)

	return &models.Week{
		Number: 0,
		Alias:  "",
		Start:  start,
		End:    end,
	}
}

func GetNextWeek(week *models.Week) *models.Week {
	return &models.Week{
		Number: week.Number + 1,
		Alias:  "",
		Start:  week.End,
		End:    week.End.Add(7 * 24 * time.Hour),
	}
}
