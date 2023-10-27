package services

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/utils"
)

type TrackService struct {
	cli             *mongo.Client
	paymentColl     *mongo.Collection
	statColl        *mongo.Collection
	weeklyTrackColl *mongo.Collection
	actionColl      *mongo.Collection
	userTrackColl   *mongo.Collection
	closeC          chan struct{}
}

func NewTrackService(cli *mongo.Client) *TrackService {
	return &TrackService{
		cli:             cli,
		paymentColl:     cli.Database(db.GetDatabaseName()).Collection("payments"),
		statColl:        cli.Database(DatabaseOfUserStatistics).Collection("user_stats"),
		weeklyTrackColl: cli.Database(DatabaseOfUserStatistics).Collection("weekly_track"),
		actionColl:      cli.Database(DatabaseOfUserAnalytics).Collection("user_actions"),
		userTrackColl:   cli.Database(DatabaseOfUserAnalytics).Collection("user_tracks"),
		closeC:          make(chan struct{}),
	}
}

func (ts *TrackService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("update user action track routine exit")
		for {
			select {
			case <-ts.closeC:
				log.Info(ctx).Msg("track service stopped.")
				return
			case <-ticker.C:
				now := time.Now()
				if now.Hour() == 0 {
					ts.weeklyViewPriceUserTracking(ctx, now)
				}
			}
		}
	}()
	return nil
}

func (ts *TrackService) weeklyViewPriceUserTracking(ctx context.Context, now time.Time) error {
	log.Info(ctx).Msgf("start stat weekly view price user tracking at: %+v\n", now)
	week := TimeToWeek(now)
	query := bson.M{
		"tag":  ActionTypeOfRedirectChangePlan,
		"time": week.Start,
	}
	cnt, err := ts.userTrackColl.CountDocuments(ctx, query)
	if err != nil {
		return err
	}
	if cnt == 0 {
		return nil
	}
	cursor, err := ts.userTrackColl.Find(ctx, query)
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	var (
		userNum    int64
		loginNum   int64
		hkbNum     int64
		vpriceNum  int64
		premiumNum int64
	)
	for cursor.Next(ctx) {
		track := &models.Track{}
		if err = cursor.Decode(track); err != nil {
			return db.HandleDBError(err)
		}
		userNum += 1
		if ts.checkLogin(ctx, track.User, week.Start) {
			loginNum += 1
		}
		if ts.checkKnowledgeBase(ctx, track.User) {
			hkbNum += 1
		}
		if ts.checkViewPrice(ctx, track.User, week.Start) {
			vpriceNum += 1
		}
		if ts.checkPay(ctx, track.User, week.Start) {
			premiumNum += 1
		}
	}
	userTrack := &models.WeeklyUserTrack{
		Week:                 week,
		Tag:                  ActionTypeOfRedirectChangePlan,
		UserNum:              userNum,
		LoginNum:             loginNum,
		HighKnowledgeBaseNum: hkbNum,
		ViewPriceNum:         vpriceNum,
		PremiumNum:           premiumNum,
	}
	queryWeeklyUserTrack := bson.M{
		"week.alias": week.Alias,
		"tag":        ActionTypeOfRedirectChangePlan,
	}
	opts := &options.ReplaceOptions{
		Upsert: utils.PtrBool(true),
	}
	_, err = ts.weeklyTrackColl.ReplaceOne(ctx, queryWeeklyUserTrack, userTrack, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to insert daily stat")
		return err
	}
	log.Info(ctx).Msgf("finish stat weekly view price user at: %+v\n", time.Now())
	return nil
}

func (ts *TrackService) checkLogin(ctx context.Context, oid string, start time.Time) bool {
	query := bson.M{
		"usersub": oid,
		"time": bson.M{
			"$gte": start.Format(time.RFC3339),
		},
	}
	result := ts.actionColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return false
		}
		log.Error(ctx).Err(result.Err()).Str("user", oid).Msg("failed to check user login")
		return false
	}
	return true
}

func (ts *TrackService) checkViewPrice(ctx context.Context, oid string, start time.Time) bool {
	query := bson.M{
		"usersub": oid,
		"action":  ActionTypeOfRedirectChangePlan,
		"time": bson.M{
			"$gte": start.Format(time.RFC3339),
		},
	}
	result := ts.actionColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return false
		}
		log.Error(ctx).Err(result.Err()).Str("user", oid).Msg("failed to check user view price")
		return false
	}
	return true
}

func (ts *TrackService) checkPay(ctx context.Context, oid string, start time.Time) bool {
	query := bson.M{
		"created_by": oid,
		"created_at": bson.M{
			"$gte": start,
		},
	}
	result := ts.paymentColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return false
		}
		log.Error(ctx).Err(result.Err()).Str("user", oid).Msg("failed to check user pay")
		return false
	}
	return true
}

func (ts *TrackService) checkKnowledgeBase(ctx context.Context, oid string) bool {
	query := bson.M{
		"oidc_id": oid,
	}
	result := ts.statColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return false
		}
		log.Error(ctx).Err(result.Err()).Str("user", oid).Msg("failed to check knowledge base")
		return false
	}
	user := &models.User{}
	if err := result.Decode(user); err != nil {
		log.Error(ctx).Err(err).Str("user", oid).Msg("failed to decode user")
		return false
	}
	return user.Usages.AI.KnowledgeBase > 0
}

func (ts *TrackService) getViewPriceUsers(ctx context.Context, now time.Time) ([]string, map[string]uint64, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"action", "redirect_change_plan"},
				{"website", bson.M{
					"$ne": "https://ai.vanustest.com",
				}},
				{"time", bson.M{
					"$gte": now.UTC().Add(-1 * TimeDurationOfWeek).Format(time.RFC3339),
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$usersub"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
		{
			{"$sort", bson.D{
				{"count", -1},
			}},
		},
	}
	type countGroup struct {
		UserID string `bson:"_id"`
		Count  uint64 `bson:"count"`
	}
	cursor, err := ts.actionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		return nil, nil, err
	}
	defer cursor.Close(ctx)
	users := make([]string, 0)
	counts := make(map[string]uint64)
	for cursor.Next(ctx) {
		var countGroup countGroup
		if err = cursor.Decode(&countGroup); err != nil {
			return nil, nil, err
		}
		users = append(users, countGroup.UserID)
		counts[countGroup.UserID] = countGroup.Count
	}
	return users, counts, nil
}

func (ts *TrackService) Stop() error {
	return nil
}
