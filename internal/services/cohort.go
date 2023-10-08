package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
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

var (
	StartAt time.Time = time.Date(2023, 3, 9, 0, 0, 0, 0, time.UTC)
)

type CohortService struct {
	mgoCli     *mongo.Client
	statColl   *mongo.Collection
	cohortColl *mongo.Collection
	wg         sync.WaitGroup
	closeC     chan struct{}
}

func NewCohortService(cli *mongo.Client) *CohortService {
	return &CohortService{
		mgoCli:     cli,
		statColl:   cli.Database(db.GetDatabaseName()).Collection("stats"),
		cohortColl: cli.Database(db.GetDatabaseName()).Collection("weekly_cohort"),
		closeC:     make(chan struct{}),
	}
}

func (cs *CohortService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("weekly cohort analysis routine exit")
		for {
			select {
			case <-cs.closeC:
				log.Info(ctx).Msg("weekly cohort analysis service stopped.")
				return
			case <-ticker.C:
				now := time.Now()
				if now.Weekday() == time.Monday && now.Hour() == 2 {
					log.Info(ctx).Msgf("start weekly cohort analysis at: %+v\n", now)
					err := cs.WeeklyCohortAnalysis(ctx, now)
					if err != nil {
						log.Error(ctx).Msgf("weekly cohort analysis failed at %+v\n", time.Now())
					}
					log.Info(ctx).Msgf("finish weekly cohort analysis at: %+v\n", time.Now())
				}
			}
		}
	}()
	return nil
}

func (cs *CohortService) Stop() error {
	return nil
}

func (cs *CohortService) WeeklyCohortAnalysis(ctx context.Context, now time.Time) error {
	goroutines := 0
	week := TimeToWeek(StartAt)
	for {
		cs.wg.Add(1)
		goroutines += 1
		go cs.updateOrInsertWeeklyCohortAnalysis(ctx, week)
		// log.Info(ctx).Int("goroutine cnt", goroutines).Str("week start", week.Alias).Msgf("start a goroutine to update")
		week = GetNextWeek(week)
		if week.End.After(now) {
			break
		}
	}
	log.Info(ctx).Msgf("launch a total of %d goroutines, with each goroutine assigned a weekly cohort analysis tasks\n", goroutines)
	log.Info(ctx).Msg("starting weekly cohort analysis, please wait...")
	cs.wg.Wait()
	log.Info(ctx).Msgf("finished all weekly cohort analysis, spent %f seconds, updated %d weekly cohort analysis data\n", time.Since(now).Seconds(), goroutines)
	return nil
}

func (cs *CohortService) updateOrInsertWeeklyCohortAnalysis(ctx context.Context, week *models.Week) {
	defer cs.wg.Done()
	aiRetentions, ctRetentions, cnt, err := cs.getWeeklyRetentions(ctx, week)
	if err != nil && err != mongo.ErrNoDocuments {
		log.Error(ctx).Err(err).Str("week", week.Alias).Msg("failed to get weekly retentions")
		return
	}
	weeklyCohortAnalysis, exist, err := cs.getWeeklyCohortAnalysisFromAlias(ctx, week.Alias)
	if err != nil {
		log.Error(ctx).Err(err).Str("week", week.Alias).Msg("failed to weekly cohort analysis")
		return
	}
	if exist {
		weeklyCohortAnalysis.UpdatedAt = time.Now()
	} else {
		weeklyCohortAnalysis.Base = cloud.NewBase(ctx)
	}
	weeklyCohortAnalysis.Week = week
	weeklyCohortAnalysis.TotalUsers = cnt
	weeklyCohortAnalysis.AIRetention = aiRetentions
	weeklyCohortAnalysis.CTRetention = ctRetentions
	query := bson.M{
		"_id": weeklyCohortAnalysis.ID,
	}
	opts := &options.ReplaceOptions{
		Upsert: utils.PtrBool(true),
	}
	_, err = cs.cohortColl.ReplaceOne(ctx, query, weeklyCohortAnalysis, opts)
	if err != nil {
		log.Error(ctx).Err(err).Str("week", week.Alias).Msg("failed to replace weekly cohort analysis")
		return
	}
}

func (cs *CohortService) getWeeklyRetentions(ctx context.Context, week *models.Week) (map[string]*models.WeeklyRetention, map[string]*models.WeeklyRetention, uint64, error) {
	totalUsers := uint64(0)
	aiWeeklyActiveUsers := make(map[string]uint64)
	ctWeeklyActiveUsers := make(map[string]uint64)
	aiRetentions := make(map[string]*models.WeeklyRetention)
	ctRetentions := make(map[string]*models.WeeklyRetention)
	query := bson.M{
		"created_at": bson.M{
			"$gte": week.Start,
			"$lte": week.End,
		},
	}
	cursor, err := cs.statColl.Find(ctx, query)
	if err != nil {
		return aiRetentions, ctRetentions, 0, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, nil, 0, err
		}
		totalUsers += 1
		for weekNum, retention := range user.Cohort.AI {
			if _, ok := aiRetentions[weekNum]; ok {
				aiRetentions[weekNum].Usage += retention.Usage
			} else {
				aiRetentions[weekNum] = &models.WeeklyRetention{
					Week:  retention.Week,
					Usage: retention.Usage,
				}
			}
			if retention.Active {
				aiWeeklyActiveUsers[weekNum] += 1
			}
		}
		for weekNum, retention := range user.Cohort.Connect {
			if _, ok := ctRetentions[weekNum]; ok {
				ctRetentions[weekNum].Usage += retention.Usage
			} else {
				ctRetentions[weekNum] = &models.WeeklyRetention{
					Week:  retention.Week,
					Usage: retention.Usage,
				}
			}
			if retention.Active {
				ctWeeklyActiveUsers[weekNum] += 1
			}
		}
		// log.Info(ctx).Uint64("cnt", cnt).Any("user", user).Msg("success to get user")
	}
	for weekNum, activeNum := range aiWeeklyActiveUsers {
		ratio := math.Round(float64(activeNum)/float64(totalUsers)*10000) / 100
		aiRetentions[weekNum].Ratio = fmt.Sprintf("%0.2f%%", ratio)
	}
	for weekNum, activeNum := range ctWeeklyActiveUsers {
		ratio := math.Round(float64(activeNum)/float64(totalUsers)*10000) / 100
		ctRetentions[weekNum].Ratio = fmt.Sprintf("%0.2f%%", ratio)
	}
	return aiRetentions, ctRetentions, totalUsers, nil
}

func (cs *CohortService) getWeeklyCohortAnalysisFromAlias(ctx context.Context, alias string) (*models.WeeklyCohortAnalysis, bool, error) {
	weeklyCohortAnalysis := &models.WeeklyCohortAnalysis{}
	query := bson.M{
		"week.alias": alias,
	}
	result := cs.cohortColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return weeklyCohortAnalysis, false, nil
		}
		log.Error(ctx).Err(result.Err()).Str("week", alias).Msg("failed to get weekly cohort analysis")
		return nil, false, result.Err()
	}
	if err := result.Decode(weeklyCohortAnalysis); err != nil {
		log.Error(ctx).Err(err).Str("week", alias).Msg("failed to weekly cohort analysis")
		return nil, true, err
	}
	return weeklyCohortAnalysis, true, nil
}

// func (cas *CohortService) GetRetentions(ctx context.Context, week *models.Week, users []*models.User) ([]*models.Retention, error) {
// 	query := bson.M{}
// 	cnt, err := cas.statColl.CountDocuments(ctx, query)
// 	if err != nil {
// 		return nil, err
// 	}
// 	log.Info(ctx).Msgf("current collection time is %+v, with a total of %d users\n", time.Now(), cnt)
// 	step := int64(500)
// 	goroutines := 0
// 	for i := int64(0); i < cnt; {
// 		// start := i
// 		end := i + step
// 		if end > cnt {
// 			end = cnt
// 		}
// 		cas.wg.Add(1)
// 		goroutines += 1
// 		// go cas.rangeRefresh(ctx, start, end, now)
// 		i += step
// 	}
// 	log.Info(ctx).Msgf("launch a total of %d goroutines, with each goroutine assigned %d user collection tasks\n", goroutines, step)
// 	log.Info(ctx).Msg("starting collection, please wait...")
// 	cas.wg.Wait()
// 	return nil, nil
// }

// func (cas *CohortService) GetUsersFromWeek(ctx context.Context, week *models.Week) ([]*models.User, error) {
// 	users := make([]*models.User, 0)
// 	query := bson.M{
// 		"created_at": bson.M{
// 			"$gte": week.Start,
// 			"$lte": week.End,
// 		},
// 	}
// 	cursor, err := cas.statColl.Find(ctx, query)
// 	if err != nil {
// 		if err == mongo.ErrNoDocuments {
// 			log.Warn(ctx).Err(err).Msg("find stat but no documents")
// 			return users, nil
// 		}
// 		return nil, err
// 	}
// 	defer func() {
// 		_ = cursor.Close(ctx)
// 	}()
// 	for cursor.Next(ctx) {
// 		user := &models.User{}
// 		if err = cursor.Decode(user); err != nil {
// 			return nil, err
// 		}
// 		users = append(users, user)
// 	}
// 	return users, nil
// }
// func (cs *CohortService) GetCohortFromWeek(ctx context.Context, week *models.Week) (*models.Cohort, bool, error) {
// 	query := bson.M{
// 		"week.number": week.Number,
// 	}
// 	result := cs.cohortColl.FindOne(ctx, query)
// 	if result.Err() != nil {
// 		if result.Err() == mongo.ErrNoDocuments {
// 			return nil, false, nil
// 		}
// 	}
// 	cohort := &models.Cohort{}
// 	if err := result.Decode(cohort); err != nil {
// 		log.Error(ctx).Err(err).Msg("failed to decode user quota")
// 		return nil, false, db.HandleDBError(err)
// 	}
// 	return cohort, true, nil
// }

func TimeToWeek(at time.Time) *models.Week {
	// 获取当前时间所在的周一和周日的时间
	weekday := at.Weekday()
	startOffset := time.Duration((weekday + 6) % 7)
	// endOffset := time.Duration((7 - weekday) % 7)

	start := at.Add(-startOffset * 24 * time.Hour).Truncate(24 * time.Hour)
	// end := at.Add(endOffset * 24 * time.Hour).Truncate(24 * time.Hour)
	end := start.Add(TimeDurationOfWeek)

	return &models.Week{
		Number: 0,
		Alias:  fmt.Sprintf("%s %d, %d", start.Month().String(), start.Day(), start.Year()),
		Start:  start,
		End:    end,
	}
}

func GetNextWeek(week *models.Week) *models.Week {
	var alias string
	if strings.HasPrefix(week.Alias, "week") {
		alias = fmt.Sprintf("week %02d", week.Number+1)
	} else {
		alias = fmt.Sprintf("%s %d, %d", week.End.Month().String(), week.End.Day(), week.End.Year())
	}
	return &models.Week{
		Number: week.Number + 1,
		Alias:  alias,
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
	for {
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
