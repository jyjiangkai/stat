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
	"github.com/jyjiangkai/stat/plausible"
	"github.com/jyjiangkai/stat/utils"
)

const (
	BatchSize                                         = 2000
	DatabaseOfUserStatistics                          = "vanus-user-statistics"
	UserActionOfShopifyLandingPage                    = "user_action_of_shopify_landing_page"
	UserActionOfGithubLandingPage                     = "user_action_of_github_landing_page"
	UserActionOfAWSCampaignsPage                      = "user_action_of_aws_compaigns_page"
	TemplateOfShopifyWebhookToGoogleSheets_20231023_1 = "shopify-webhook-google-sheets-20231023_1"
	TemplateOfShopifyWebhookToGoogleSheets_20231023_2 = "shopify-webhook-google-sheets-20231023_2"
	TemplateOfShopifyWebhookToMailchimp_20231020_2    = "shopify-webhook-mailchimp-20231020_2"
	TemplateOfShopifyWebhookToMySQL_20231023_3        = "shopify-webhook-mysql-20231023_3"
	TemplateOfShopifyWebhookToOutlook_20231023_4      = "shopify-webhook-outlook-20231023_4"
	TemplateOfShopifyWebhookToOutlook_20231023_5      = "shopify-webhook-outlook-20231023_5"
	TemplateOfShopifyWebhookToSlack_20231023_6        = "shopify-webhook-slack-20231023_6"
	TemplateOfShopifyWebhookToSlack_20231023_7        = "shopify-webhook-slack-20231023_7"
	TemplateOfGithubToSlack_20230308_6                = "github-slack-20230308_6"
	TemplateOfGithubToSlack_20230316_3                = "github-slack-20230316_3"
	TemplateOfGithubToFeishu_20230306_1               = "github-feishu-20230306_1"
	TemplateOfGithubToFeishu_20230307_3               = "github-feishu-20230307_3"
	TemplateOfGithubToGoogleSheets_20230309_7         = "github-google-sheets-20230309_7"
	TemplateOfGithubToDiscord_20230320_3              = "github-discord-20230320_3"
	TemplateOfGithubToDiscord_20230321_1              = "github-discord-20230321_1"
)

type StatService struct {
	mgoCli              *mongo.Client
	connectionColl      *mongo.Collection
	userColl            *mongo.Collection
	quotaColl           *mongo.Collection
	paymentColl         *mongo.Collection
	billColl            *mongo.Collection
	aiBillColl          *mongo.Collection
	aiAppColl           *mongo.Collection
	aiUploadColl        *mongo.Collection
	aiKnowledgeBaseColl *mongo.Collection
	userStatColl        *mongo.Collection
	dailyStatColl       *mongo.Collection
	actionColl          *mongo.Collection
	wg                  sync.WaitGroup
	closeC              chan struct{}
}

func NewStatService(cli *mongo.Client) *StatService {
	return &StatService{
		mgoCli:              cli,
		connectionColl:      cli.Database(db.GetDatabaseName()).Collection("connections"),
		userColl:            cli.Database(db.GetDatabaseName()).Collection("users"),
		quotaColl:           cli.Database(db.GetDatabaseName()).Collection("quotas"),
		paymentColl:         cli.Database(db.GetDatabaseName()).Collection("payments"),
		billColl:            cli.Database(db.GetDatabaseName()).Collection("bills"),
		aiBillColl:          cli.Database(db.GetDatabaseName()).Collection("ai_bills"),
		aiAppColl:           cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		aiUploadColl:        cli.Database(db.GetDatabaseName()).Collection("ai_upload"),
		aiKnowledgeBaseColl: cli.Database(db.GetDatabaseName()).Collection("ai_knowledge_bases"),
		userStatColl:        cli.Database(DatabaseOfUserStatistics).Collection("user_stats"),
		dailyStatColl:       cli.Database(DatabaseOfUserStatistics).Collection("daily_stats"),
		actionColl:          cli.Database(DatabaseOfUserAnalytics).Collection("user_actions"),
		closeC:              make(chan struct{}),
	}
}

func (ss *StatService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("stat routine exit")
		for {
			select {
			case <-ss.closeC:
				log.Info(ctx).Msg("stat service stopped.")
				return
			case <-ticker.C:
				now := time.Now()
				if now.Hour() == 1 {
					log.Info(ctx).Msgf("start stat daily data at: %+v\n", now)
					err := ss.DailyStat(ctx, now)
					if err != nil {
						log.Error(ctx).Err(err).Msgf("refresh daily stat failed at %+v\n", time.Now())
					} else {
						log.Info(ctx).Msgf("finish refresh daily stat at: %+v\n", time.Now())
					}
					log.Info(ctx).Msgf("start stat user data at: %+v\n", now)
					err = ss.UserStat(ctx, now)
					if err != nil {
						log.Error(ctx).Err(err).Msgf("refresh user stat failed at %+v\n", time.Now())
					} else {
						log.Info(ctx).Msgf("finish refresh user stat at: %+v\n", time.Now())
					}
				}
			}
		}
	}()
	return nil
}

func (ss *StatService) Stop() error {
	return nil
}

func (ss *StatService) UserStat(ctx context.Context, now time.Time) error {
	query := bson.M{}
	cnt, err := ss.userColl.CountDocuments(ctx, query)
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
		ss.wg.Add(1)
		goroutines += 1
		go ss.rangeUserStat(ctx, start, end, now)
		i += step
	}
	log.Info(ctx).Msgf("launch a total of %d goroutines, with each goroutine assigned %d user collection tasks\n", goroutines, step)
	log.Info(ctx).Msg("starting collection, please wait...")
	ss.wg.Wait()
	log.Info(ctx).Msgf("finished user data collection, spent %f seconds, updated %d users\n", time.Since(now).Seconds(), cnt)
	return nil
}

func (ss *StatService) DailyStat(ctx context.Context, now time.Time) error {
	monthly := StartAt
	goroutines := 0
	for {
		ss.wg.Add(1)
		goroutines += 1
		go func(at time.Time) {
			err := ss.rangeDailyStat(ctx, at)
			if err != nil {
				log.Error(ctx).Err(err).Any("monthly", at).Msg("failed to stat range daily")
			}
		}(monthly)
		nextMonthTime := monthly.AddDate(0, 1, 0)
		monthly = time.Date(nextMonthTime.Year(), nextMonthTime.Month(), 1, 0, 0, 0, 0, time.UTC)
		if monthly.After(now) {
			break
		}
	}
	log.Info(ctx).Msgf("launch a total of %d goroutines, with each goroutine assigned a month collection tasks\n", goroutines)
	log.Info(ctx).Msg("starting daily data collection, please wait...")
	ss.wg.Wait()
	log.Info(ctx).Msgf("finished daily data collection, spent %f seconds, updated %d daily data\n", time.Since(now).Seconds(), int64(time.Since(StartAt).Hours()/24))
	return nil
}

func (ss *StatService) rangeDailyStat(ctx context.Context, date time.Time) error {
	defer ss.wg.Done()
	daily := date
	for {
		err := ss.dailyStatOfUserNumber(ctx, daily)
		if err != nil {
			return err
		}
		err = ss.dailyStatOfShopifyLandingPageActionNumber(ctx, daily)
		if err != nil {
			return err
		}
		err = ss.dailyStatOfGithubLandingPageActionNumber(ctx, daily)
		if err != nil {
			return err
		}
		err = ss.dailyStatOfAWSCampaignsPageActionNumber(ctx, daily)
		if err != nil {
			return err
		}
		daily = daily.AddDate(0, 0, 1)
		if daily.Month() == time.Now().Month() {
			if daily.Day() >= time.Now().Day() || daily.Month() > date.Month() {
				break
			}
		} else {
			if daily.Month() > date.Month() {
				break
			}
		}
	}
	return nil
}

func (ss *StatService) dailyStatOfUserNumber(ctx context.Context, date time.Time) error {
	startAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 0, 1)
	registerUserNumber, err := ss.GetRegisterUserNumber(ctx, startAt, endAt)
	if err != nil {
		return err
	}
	loginUserNumber, err := ss.GetLoginUserNumber(ctx, startAt, endAt)
	if err != nil {
		return err
	}
	cnCreatedUserNumber, err := ss.GetConnectionCreatedUserNumber(ctx, startAt, endAt)
	if err != nil {
		return err
	}
	appCreatedUserNumber, err := ss.GetAppCreatedUserNumber(ctx, startAt, endAt)
	if err != nil {
		return err
	}
	cnUsedUserNumber, err := ss.GetConnectionUsedUserNumber(ctx, startAt, endAt)
	if err != nil {
		return err
	}
	appUsedUserNumber, err := ss.GetAppUsedUserNumber(ctx, startAt, endAt)
	if err != nil {
		return err
	}
	daily := &models.DailyStatsOfUserNumber{
		Date:                        startAt,
		Tag:                         "user_number",
		RegisterUserNumber:          registerUserNumber,
		LoginUserNumber:             loginUserNumber,
		ConnectionCreatedUserNumber: cnCreatedUserNumber,
		AppCreatedUserNumber:        appCreatedUserNumber,
		ConnectionUsedUserNumber:    cnUsedUserNumber,
		AppUsedUserNumber:           appUsedUserNumber,
	}
	query := bson.M{
		"date": startAt,
		"tag":  "user_number",
	}
	opts := &options.ReplaceOptions{
		Upsert: utils.PtrBool(true),
	}
	_, err = ss.dailyStatColl.ReplaceOne(ctx, query, daily, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to insert daily user number stat")
		return err
	}
	// log.Info(ctx).Any("date", startAt).Msg("finished daily user number stat")
	return nil
}

func (ss *StatService) dailyStatOfShopifyLandingPageActionNumber(ctx context.Context, date time.Time) error {
	if date.Before(time.Date(2023, 10, 26, 0, 0, 0, 0, time.UTC)) {
		return nil
	}
	startAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 0, 1)
	visitors, err := plausible.GetVisitors(ctx, "vanus.ai", "/connectors/shopify", startAt.Format("2006-01-02"))
	if err != nil {
		return err
	}
	tryVanusActionNumber, err := ss.GetTryVanusActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage)
	if err != nil {
		return err
	}
	signInWithGithubActionNumber, err := ss.GetSignInActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, "github-auth")
	if err != nil {
		return err
	}
	signInWithGoogleActionNumber, err := ss.GetSignInActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, "google-auth")
	if err != nil {
		return err
	}
	signInWithMicrosoftActionNumber, err := ss.GetSignInActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, "microsoft-auth")
	if err != nil {
		return err
	}
	ContactUsActionNumber, err := ss.GetContactUsActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage)
	if err != nil {
		return err
	}
	num1, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToGoogleSheets_20231023_1)
	if err != nil {
		return err
	}
	num2, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToGoogleSheets_20231023_2)
	if err != nil {
		return err
	}
	num3, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToMailchimp_20231020_2)
	if err != nil {
		return err
	}
	num4, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToMySQL_20231023_3)
	if err != nil {
		return err
	}
	num5, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToOutlook_20231023_4)
	if err != nil {
		return err
	}
	num6, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToOutlook_20231023_5)
	if err != nil {
		return err
	}
	num7, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToSlack_20231023_6)
	if err != nil {
		return err
	}
	num8, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfShopifyLandingPage, TemplateOfShopifyWebhookToSlack_20231023_7)
	if err != nil {
		return err
	}

	daily := &models.DailyStatsOfShopifyLandingPageActionNumber{
		Date:                            startAt,
		Tag:                             UserActionOfShopifyLandingPage,
		UniqueVisitorNumber:             visitors,
		TryVanusActionNumber:            tryVanusActionNumber,
		SignInWithGithubActionNumber:    signInWithGithubActionNumber,
		SignInWithGoogleActionNumber:    signInWithGoogleActionNumber,
		SignInWithMicrosoftActionNumber: signInWithMicrosoftActionNumber,
		ContactUsActionNumber:           ContactUsActionNumber,
		ShopifyToGoogleSheetsWithNewOrderActionNumber:    num1,
		ShopifyToGoogleSheetsWithCancelOrderActionNumber: num2,
		ShopifyToMailChimpActionNumber:                   num3,
		ShopifyToMySQLActionNumber:                       num4,
		ShopifyToOutlookWithWelcomeCustomerActionNumber:  num5,
		ShopifyToOutlookWithNewOrderActionNumber:         num6,
		ShopifyToSlackWithNewOrderActionNumber:           num7,
		ShopifyToSlackWithCancelOrderActionNumber:        num8,
	}
	query := bson.M{
		"date": startAt,
		"tag":  UserActionOfShopifyLandingPage,
	}
	opts := &options.ReplaceOptions{
		Upsert: utils.PtrBool(true),
	}
	_, err = ss.dailyStatColl.ReplaceOne(ctx, query, daily, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to insert daily action of shopify landing page")
		return err
	}
	// log.Info(ctx).Any("date", startAt).Msg("finished daily user action of shopify stat")
	return nil
}

func (ss *StatService) dailyStatOfGithubLandingPageActionNumber(ctx context.Context, date time.Time) error {
	if date.Before(time.Date(2023, 10, 26, 0, 0, 0, 0, time.UTC)) {
		return nil
	}
	startAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endAt := startAt.AddDate(0, 0, 1)
	visitors, err := plausible.GetVisitors(ctx, "vanus.ai", "/connectors/github", startAt.Format("2006-01-02"))
	if err != nil {
		return err
	}
	tryVanusActionNumber, err := ss.GetTryVanusActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage)
	if err != nil {
		return err
	}
	signInWithGithubActionNumber, err := ss.GetSignInActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, "github-auth")
	if err != nil {
		return err
	}
	signInWithGoogleActionNumber, err := ss.GetSignInActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, "google-auth")
	if err != nil {
		return err
	}
	signInWithMicrosoftActionNumber, err := ss.GetSignInActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, "microsoft-auth")
	if err != nil {
		return err
	}
	ContactUsActionNumber, err := ss.GetContactUsActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage)
	if err != nil {
		return err
	}
	num1, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, TemplateOfGithubToSlack_20230308_6)
	if err != nil {
		return err
	}
	num2, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, TemplateOfGithubToSlack_20230316_3)
	if err != nil {
		return err
	}
	num3, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, TemplateOfGithubToFeishu_20230306_1)
	if err != nil {
		return err
	}
	num4, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, TemplateOfGithubToFeishu_20230307_3)
	if err != nil {
		return err
	}
	num5, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, TemplateOfGithubToGoogleSheets_20230309_7)
	if err != nil {
		return err
	}
	num6, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, TemplateOfGithubToDiscord_20230320_3)
	if err != nil {
		return err
	}
	num7, err := ss.GetTemplateTryItActionNumber(ctx, startAt, endAt, UserActionOfGithubLandingPage, TemplateOfGithubToDiscord_20230321_1)
	if err != nil {
		return err
	}
	daily := &models.DailyStatsOfGithubLandingPageActionNumber{
		Date:                                        startAt,
		Tag:                                         UserActionOfGithubLandingPage,
		UniqueVisitorNumber:                         visitors,
		TryVanusActionNumber:                        tryVanusActionNumber,
		SignInWithGithubActionNumber:                signInWithGithubActionNumber,
		SignInWithGoogleActionNumber:                signInWithGoogleActionNumber,
		SignInWithMicrosoftActionNumber:             signInWithMicrosoftActionNumber,
		ContactUsActionNumber:                       ContactUsActionNumber,
		GithubToSlackWithIssueActionNumber:          num1,
		GithubToSlackWithOpenedPRActionNumber:       num2,
		GithubToFeishuWithStarActionNumber:          num3,
		GithubToFeishuWithIssueCommentActionNumber:  num4,
		GithubToGoogleSheetsWithIssueActionNumber:   num5,
		GithubToDiscordWithIssueCommentActionNumber: num6,
		GithubToDiscordWithOpenedPRActionNumber:     num7,
	}
	query := bson.M{
		"date": startAt,
		"tag":  UserActionOfGithubLandingPage,
	}
	opts := &options.ReplaceOptions{
		Upsert: utils.PtrBool(true),
	}
	_, err = ss.dailyStatColl.ReplaceOne(ctx, query, daily, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to insert daily action of github landing page")
		return err
	}
	// log.Info(ctx).Any("date", startAt).Msg("finished daily user action of shopify stat")
	return nil
}

func (ss *StatService) dailyStatOfAWSCampaignsPageActionNumber(ctx context.Context, date time.Time) error {
	if date.Before(time.Date(2023, 10, 26, 0, 0, 0, 0, time.UTC)) {
		return nil
	}
	startAt := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	// endAt := startAt.AddDate(0, 0, 1)
	visitors, err := plausible.GetVisitors(ctx, "vanus.cn", "/campaigns/aws-smb-hub-2023-11/", startAt.Format("2006-01-02"))
	if err != nil {
		return err
	}
	daily := &models.DailyStatsOfAWSCampaignsPageActionNumber{
		Date:                            startAt,
		Tag:                             UserActionOfAWSCampaignsPage,
		UniqueVisitorNumber:             visitors,
		TryVanusActionNumber:            0,
		SignInWithGithubActionNumber:    0,
		SignInWithGoogleActionNumber:    0,
		SignInWithMicrosoftActionNumber: 0,
		ContactUsActionNumber:           0,
	}
	query := bson.M{
		"date": startAt,
		"tag":  UserActionOfAWSCampaignsPage,
	}
	opts := &options.ReplaceOptions{
		Upsert: utils.PtrBool(true),
	}
	_, err = ss.dailyStatColl.ReplaceOne(ctx, query, daily, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to insert daily action of github landing page")
		return err
	}
	// log.Info(ctx).Any("date", startAt).Msg("finished daily user action of shopify stat")
	return nil
}

func (ss *StatService) GetRegisterUserNumber(ctx context.Context, start, end time.Time) (int64, error) {
	query := bson.M{
		"created_at": bson.M{
			"$gte": start,
			"$lte": end,
		},
	}
	cnt, err := ss.userColl.CountDocuments(ctx, query)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

// TODO(jiangkai): 当前没有登入的埋点数据，只有登出的埋点数据
func (ss *StatService) GetLoginUserNumber(ctx context.Context, start, end time.Time) (int64, error) {
	if start == time.Date(2023, 9, 14, 0, 0, 0, 0, time.UTC) {
		return 517, nil
	}
	if start == time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC) {
		return 427, nil
	}
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"time", bson.M{
					"$gte": start.Format(time.RFC3339),
					"$lte": end.Format(time.RFC3339),
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$usersub"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
	}
	cursor, err := ss.actionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return 0, err
	}
	defer cursor.Close(ctx)
	var cnt int64
	type countGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	for cursor.Next(ctx) {
		var count countGroup
		if err = cursor.Decode(&count); err != nil {
			return 0, err
		}
		cnt += 1
	}
	return cnt, nil
}

func (ss *StatService) GetConnectionCreatedUserNumber(ctx context.Context, start, end time.Time) (int64, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"created_at", bson.M{
					"$gte": start,
					"$lte": end,
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$created_by"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
	}
	cursor, err := ss.connectionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return 0, err
	}
	defer cursor.Close(ctx)
	var cnt int64
	type countGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	for cursor.Next(ctx) {
		var count countGroup
		if err = cursor.Decode(&count); err != nil {
			return 0, err
		}
		cnt += 1
	}
	return cnt, nil
}

func (ss *StatService) GetAppCreatedUserNumber(ctx context.Context, start, end time.Time) (int64, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"created_at", bson.M{
					"$gte": start,
					"$lte": end,
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$created_by"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
	}
	cursor, err := ss.aiAppColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return 0, err
	}
	defer cursor.Close(ctx)
	var cnt int64
	type countGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	for cursor.Next(ctx) {
		var count countGroup
		if err = cursor.Decode(&count); err != nil {
			return 0, err
		}
		cnt += 1
	}
	return cnt, nil
}

func (ss *StatService) GetConnectionUsedUserNumber(ctx context.Context, start, end time.Time) (int64, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"collected_at", end},
				{"delivered_num", bson.M{
					"$ne": 0,
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$user_id"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
	}
	cursor, err := ss.billColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return 0, err
	}
	defer cursor.Close(ctx)
	var cnt int64
	type countGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	for cursor.Next(ctx) {
		var count countGroup
		if err = cursor.Decode(&count); err != nil {
			return 0, err
		}
		cnt += 1
	}
	return cnt, nil
}

func (ss *StatService) GetAppUsedUserNumber(ctx context.Context, start, end time.Time) (int64, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"collected_at", bson.M{
					"$gte": start,
					"$lt":  end,
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$user_id"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
	}
	cursor, err := ss.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return 0, err
	}
	defer cursor.Close(ctx)
	var cnt int64
	type countGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	for cursor.Next(ctx) {
		var count countGroup
		if err = cursor.Decode(&count); err != nil {
			return 0, err
		}
		cnt += 1
	}
	return cnt, nil
}

func (ss *StatService) GetTryVanusActionNumber(ctx context.Context, start, end time.Time, tag string) (int64, error) {
	if start == time.Date(2023, 9, 14, 0, 0, 0, 0, time.UTC) {
		return 517, nil
	}
	if start == time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC) {
		return 427, nil
	}
	from := ""
	if tag == UserActionOfShopifyLandingPage {
		from = "https://www.vanus.ai/connectors/shopify"
	} else if tag == UserActionOfGithubLandingPage {
		from = "https://www.vanus.ai/connectors/github"
	}
	query := bson.M{
		"time": bson.M{
			"$gte": start.Format(time.RFC3339),
			"$lte": end.Format(time.RFC3339),
		},
		"action": "redirect_login",
		"source": "www.vanus.ai",
		"payload.from": bson.M{
			"$regex": from,
		},
		"payload.extra": bson.M{
			"$exists": false,
		},
	}
	cnt, err := ss.actionColl.CountDocuments(ctx, query)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (ss *StatService) GetSignInActionNumber(ctx context.Context, start, end time.Time, tag string, auth string) (int64, error) {
	if start == time.Date(2023, 9, 14, 0, 0, 0, 0, time.UTC) {
		return 517, nil
	}
	if start == time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC) {
		return 427, nil
	}
	from := ""
	if tag == UserActionOfShopifyLandingPage {
		from = "https://www.vanus.ai/connectors/shopify"
	} else if tag == UserActionOfGithubLandingPage {
		from = "https://www.vanus.ai/connectors/github"
	}
	query := bson.M{
		"time": bson.M{
			"$gte": start.Format(time.RFC3339),
			"$lte": end.Format(time.RFC3339),
		},
		"action": "redirect_login",
		"source": "www.vanus.ai",
		"payload.from": bson.M{
			"$regex": from,
		},
		"payload.extra": auth,
	}
	cnt, err := ss.actionColl.CountDocuments(ctx, query)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (ss *StatService) GetContactUsActionNumber(ctx context.Context, start, end time.Time, tag string) (int64, error) {
	if start == time.Date(2023, 9, 14, 0, 0, 0, 0, time.UTC) {
		return 517, nil
	}
	if start == time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC) {
		return 427, nil
	}
	from := ""
	if tag == UserActionOfShopifyLandingPage {
		from = "https://www.vanus.ai/connectors/shopify"
	} else if tag == UserActionOfGithubLandingPage {
		from = "https://www.vanus.ai/connectors/github"
	}
	query := bson.M{
		"time": bson.M{
			"$gte": start.Format(time.RFC3339),
			"$lte": end.Format(time.RFC3339),
		},
		"action": "contact",
		"payload.from": bson.M{
			"$regex": from,
		},
		"source": "www.vanus.ai",
	}
	cnt, err := ss.actionColl.CountDocuments(ctx, query)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (ss *StatService) GetTemplateTryItActionNumber(ctx context.Context, start, end time.Time, tag string, template string) (int64, error) {
	if start == time.Date(2023, 9, 14, 0, 0, 0, 0, time.UTC) {
		return 517, nil
	}
	if start == time.Date(2023, 9, 15, 0, 0, 0, 0, time.UTC) {
		return 427, nil
	}
	from := ""
	if tag == UserActionOfShopifyLandingPage {
		from = "https://www.vanus.ai/connectors/shopify"
	} else if tag == UserActionOfGithubLandingPage {
		from = "https://www.vanus.ai/connectors/github"
	}
	query := bson.M{
		"time": bson.M{
			"$gte": start.Format(time.RFC3339),
			"$lte": end.Format(time.RFC3339),
		},
		"action": "create_connection_from_landing",
		"source": "www.vanus.ai",
		"payload.from": bson.M{
			"$regex": from,
		},
		"template": template,
	}
	cnt, err := ss.actionColl.CountDocuments(ctx, query)
	if err != nil {
		return 0, err
	}
	return cnt, nil
}

func (ss *StatService) rangeUserStat(ctx context.Context, start int64, end int64, now time.Time) {
	var (
		reterr error
		cnt    int    = 0
		skip   int64  = start
		limit  int64  = end - start
		sort   bson.M = bson.M{"created_at": 1}
	)
	// log.Info(ctx).Msgf("start goroutine for range refresh, start: %d, end: %d\n", start, end)
	defer ss.wg.Done()
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
	cursor, err := ss.userColl.Find(ctx, query, &opt)
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
		bills, err := ss.getBills(ctx, user.OID, now)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to get bills")
			return
		}
		class, err := ss.getClass(ctx, user.OID, now)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to get class")
			return
		}
		usage, err := ss.getUsages(ctx, user.OID)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to get usages")
			return
		}
		cohort, err := ss.GetCohort(ctx, user)
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
			Industry:     ss.GetUserIndustry(ctx, user),
			Ref:          user.Ref,
			RefHost:      user.RefHost,
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
		_, err = ss.userStatColl.ReplaceOne(ctx, query, statUser, opts)
		if err != nil {
			reterr = err
			log.Error(ctx).Err(err).Msg("failed to insert stat user")
			return
		}
		// log.Info(ctx).Msgf("[%d] spent %d ms to refresh user stat: %s\n", cnt, time.Since(start).Milliseconds(), user.OID)
	}
}

func (ss *StatService) getLastStatTime(ctx context.Context) (time.Time, error) {
	var (
		sortBy string    = "updated_at"
		now    time.Time = time.Now()
	)
	query := bson.M{}
	opt := options.FindOneOptions{
		Sort: bson.M{sortBy: -1},
	}
	result := ss.userStatColl.FindOne(ctx, query, &opt)
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

func (ss *StatService) GetAIBillChangedUserList(ctx context.Context, start, end time.Time) ([]string, error) {
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
	cursor, err := ss.aiBillColl.Aggregate(ctx, pipeline)
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

func (ss *StatService) GetConnectBillChangedUserList(ctx context.Context, start, end time.Time) ([]string, error) {
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
					{"$sum", "$delivered_num"},
				}},
			}},
		},
	}
	type usageGroup struct {
		UserID string `bson:"_id"`
		Usage  uint64 `bson:"usage"`
	}
	cursor, err := ss.billColl.Aggregate(ctx, pipeline)
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

func (ss *StatService) GetUserIndustry(ctx context.Context, user *cloud.User) string {
	if user.Industry == "Others" {
		return user.IndustryExtra
	}
	return user.Industry
}
