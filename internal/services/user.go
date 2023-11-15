package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jyjiangkai/stat/api"
	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/mailchimp"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
)

const (
	UserTypeOfRegister                       = "register"
	UserTypeOfLogin                          = "login"
	UserTypeOfCreated                        = "created"
	UserTypeOfUsed                           = "used"
	UserTypeOfRegisterFromShopifyLandingPage = "register_from_shopify_landing_page"
	UserTypeOfRegisterFromGithubLandingPage  = "register_from_github_landing_page"
	UserTypeOfConnectionTemplateCreated      = "connection_template_created"
	UserTypeOfPremium                        = "premium"
	UserTypeOfNoKnownledgeBase               = "no_knowledge_base"
	UserTypeOfHighKnownledgeBase             = "high_knowledge_base"
	UserTypeOfCohort                         = "cohort"
	UserTypeOfDailyUserNumber                = "daily_user_number"
)

var (
	TimeDurationOfWeek  time.Duration = 7 * 24 * time.Hour
	TimeDurationOfMonth time.Duration = 30 * 24 * time.Hour
)

type UserService struct {
	cli                 *mongo.Client
	userColl            *mongo.Collection
	appColl             *mongo.Collection
	billColl            *mongo.Collection
	aiBillColl          *mongo.Collection
	promptColl          *mongo.Collection
	uploadColl          *mongo.Collection
	aiKnowledgeBaseColl *mongo.Collection
	connectorColl       *mongo.Collection
	connectionColl      *mongo.Collection
	userStatColl        *mongo.Collection
	dailyStatColl       *mongo.Collection
	cohortColl          *mongo.Collection
	creditColl          *mongo.Collection
	actionColl          *mongo.Collection
	trackColl           *mongo.Collection
	closeC              chan struct{}
}

func NewUserService(cli *mongo.Client) *UserService {
	return &UserService{
		cli:                 cli,
		userColl:            cli.Database(db.GetDatabaseName()).Collection("users"),
		appColl:             cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		billColl:            cli.Database(db.GetDatabaseName()).Collection("bills"),
		aiBillColl:          cli.Database(db.GetDatabaseName()).Collection("ai_bills"),
		promptColl:          cli.Database(db.GetDatabaseName()).Collection("ai_prompt"),
		uploadColl:          cli.Database(db.GetDatabaseName()).Collection("ai_upload"),
		aiKnowledgeBaseColl: cli.Database(db.GetDatabaseName()).Collection("ai_knowledge_bases"),
		connectorColl:       cli.Database(db.GetDatabaseName()).Collection("connectors"),
		connectionColl:      cli.Database(db.GetDatabaseName()).Collection("connections"),
		creditColl:          cli.Database(db.GetDatabaseName()).Collection("credits"),
		userStatColl:        cli.Database(DatabaseOfUserStatistics).Collection("user_stats"),
		dailyStatColl:       cli.Database(DatabaseOfUserStatistics).Collection("daily_stats"),
		cohortColl:          cli.Database(DatabaseOfUserStatistics).Collection("weekly_cohort"),
		actionColl:          cli.Database(DatabaseOfUserAnalytics).Collection("user_actions"),
		trackColl:           cli.Database(DatabaseOfUserAnalytics).Collection("user_tracks"),
		closeC:              make(chan struct{}),
	}
}

func (us *UserService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("weekly user tracking routine exit")
		for {
			select {
			case <-us.closeC:
				log.Info(ctx).Msg("user service stopped.")
				return
			case <-ticker.C:
				now := time.Now()
				if now.Weekday() == time.Monday && now.Hour() == 2 {
					us.weeklyNoKnownledgeBaseUserTracking(ctx, now)
					us.weeklyHighKnownledgeBaseUserTracking(ctx, now)
				}
			}
		}
	}()
	return nil
}

func (us *UserService) weeklyNoKnownledgeBaseUserTracking(ctx context.Context, now time.Time) error {
	log.Info(ctx).Msgf("start stat weekly no knownledge base user at: %+v\n", now)
	pg := api.Page{}
	req := api.NewRequest()
	opts := &api.ListOptions{
		KindSelector: "ai",
		TypeSelector: UserTypeOfNoKnownledgeBase,
	}
	result, err := us.List(ctx, pg, req, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msgf("stat weekly no knownledge base user failed at %+v\n", now)
		return err
	}
	cnt := 0
	for idx := range result.List {
		user := result.List[idx].(*models.User)
		track := &models.Track{
			User:  user.OID,
			Tag:   UserTypeOfNoKnownledgeBase,
			Count: 0,
			Time:  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		}
		_, err := us.trackColl.InsertOne(ctx, track)
		if err != nil {
			log.Error(ctx).Err(err).Any("track", track).Msg("failed to insert user track")
			return db.HandleDBError(err)
		}
		if mailchimp.ValidateEmail(user.Email) {
			tags := []string{"vanus_ai", UserTypeOfNoKnownledgeBase}
			err := mailchimp.AddMember(ctx, user.Email, tags)
			if err != nil {
				log.Error(ctx).Str("email", user.Email).Msg("failed to add member to mailchimp")
			}
		}
		cnt += 1
	}
	log.Info(ctx).Int("cnt", cnt).Msgf("finish stat weekly no knownledge base user at: %+v\n", time.Now())
	return nil
}

func (us *UserService) weeklyHighKnownledgeBaseUserTracking(ctx context.Context, now time.Time) error {
	log.Info(ctx).Msgf("start stat weekly high knownledge base user at: %+v\n", now)
	pg := api.Page{}
	req := api.NewRequest()
	opts := &api.ListOptions{
		KindSelector: "ai",
		TypeSelector: UserTypeOfHighKnownledgeBase,
	}
	result, err := us.List(ctx, pg, req, opts)
	if err != nil {
		log.Error(ctx).Err(err).Msgf("stat weekly high knownledge base user failed at %+v\n", now)
		return err
	}
	cnt := 0
	for idx := range result.List {
		user := result.List[idx].(*models.User)
		track := &models.Track{
			User:  user.OID,
			Tag:   UserTypeOfHighKnownledgeBase,
			Count: 0,
			Time:  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		}
		_, err := us.trackColl.InsertOne(ctx, track)
		if err != nil {
			log.Error(ctx).Err(err).Any("track", track).Msg("failed to insert user track")
			return db.HandleDBError(err)
		}
		if mailchimp.ValidateEmail(user.Email) {
			tags := []string{"vanus_ai", UserTypeOfHighKnownledgeBase}
			err := mailchimp.AddMember(ctx, user.Email, tags)
			if err != nil {
				log.Error(ctx).Str("email", user.Email).Msg("failed to add member to mailchimp")
			}
		}
		cnt += 1
	}
	log.Info(ctx).Int("cnt", cnt).Msgf("finish stat weekly high knownledge base user at: %+v\n", time.Now())
	return nil
}

func (us *UserService) Stop() error {
	return nil
}

func (us *UserService) List(ctx context.Context, pg api.Page, req api.Request, opts *api.ListOptions) (*api.ListResult, error) {
	log.Info(ctx).Any("page", pg).Any("request", req).Any("opts", opts).Msg("user service list api")
	switch opts.TypeSelector {
	case UserTypeOfRegister, UserTypeOfRegisterFromShopifyLandingPage, UserTypeOfRegisterFromGithubLandingPage, UserTypeOfLogin, UserTypeOfCreated, UserTypeOfUsed, UserTypeOfConnectionTemplateCreated:
		return us.listSpecifiedUsers(ctx, pg, req, opts)
	case UserTypeOfPremium:
		return us.listPremiumUsers(ctx, pg, req.FilterStack, opts)
	case UserTypeOfNoKnownledgeBase:
		return us.listNoKnownledgeBaseUsers(ctx, pg, opts)
	case UserTypeOfHighKnownledgeBase:
		return us.listHighKnownledgeBaseUsers(ctx, pg, opts)
	case UserTypeOfCohort:
		return us.listCohortUsers(ctx, pg, opts)
	case UserTypeOfDailyUserNumber:
		return us.listDailyUserNumber(ctx, pg, opts)
	default:
		return us.list(ctx, pg, req.FilterStack, opts)
	}
}

func (us *UserService) list(ctx context.Context, pg api.Page, filters api.FilterStack, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	query := addFilter(ctx, filters)
	if opts.KindSelector == "ai" {
		query["usages.ai.app"] = bson.M{"$ne": 0}
	} else if opts.KindSelector == "connect" {
		query["usages.connect.connection"] = bson.M{"$ne": 0}
	}
	log.Info(ctx).Any("query", query).Msg("show user list api query criteria")
	cnt, err := us.userStatColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}
	if cnt <= skip {
		return nil, api.ErrPageArgumentsTooLarge
	}

	pg.Total = cnt
	if pg.Direction == "asc" {
		sort = bson.M{pg.SortBy: 1}
	} else if pg.Direction == "desc" {
		sort = bson.M{pg.SortBy: -1}
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.userStatColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				P: pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, user)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listRegisterUsers(ctx context.Context, pg api.Page, rg api.Range, filters api.FilterStack, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	start, end, err := us.getRangeTime(ctx, rg)
	if err != nil {
		return nil, err
	}
	query := addFilter(ctx, filters)
	query["created_at"] = bson.M{
		"$gte": start,
		"$lte": end,
	}
	cnt, err := us.userColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}
	if cnt <= skip {
		return nil, api.ErrPageArgumentsTooLarge
	}

	pg.Total = cnt
	if pg.Direction == "asc" {
		sort = bson.M{pg.SortBy: 1}
	} else if pg.Direction == "desc" {
		sort = bson.M{pg.SortBy: -1}
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.userColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				P: pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, user)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listRegisterFromLandingPageUsers(ctx context.Context, pg api.Page, filters api.FilterStack, ref, date string) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	rg := api.Range{
		Start: date,
		End:   date,
	}
	start, end, err := us.getRangeTime(ctx, rg)
	if err != nil {
		return nil, err
	}
	query := addFilter(ctx, filters)
	query["ref"] = ref
	query["created_at"] = bson.M{
		"$gte": start,
		"$lte": end,
	}
	cnt, err := us.userColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}
	if cnt <= skip {
		return nil, api.ErrPageArgumentsTooLarge
	}

	pg.Total = cnt
	if pg.Direction == "asc" {
		sort = bson.M{pg.SortBy: 1}
	} else if pg.Direction == "desc" {
		sort = bson.M{pg.SortBy: -1}
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.userColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				P: pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, user)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listSpecifiedUsers(ctx context.Context, pg api.Page, req api.Request, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
		users []string
		err   error
	)

	if skip < 0 {
		skip = 0
	}

	switch opts.TypeSelector {
	case UserTypeOfRegister:
		return us.listRegisterUsers(ctx, pg, req.Range, req.FilterStack, opts)
	case UserTypeOfRegisterFromShopifyLandingPage:
		return us.listRegisterFromLandingPageUsers(ctx, pg, req.FilterStack, "shopify-landing", req.Point.X)
	case UserTypeOfRegisterFromGithubLandingPage:
		return us.listRegisterFromLandingPageUsers(ctx, pg, req.FilterStack, "github-landing", req.Point.X)
	case UserTypeOfLogin:
		users, err = us.getLoginUsers(ctx, req.Range)
		if err != nil {
			log.Error(ctx).Err(err).Msg("failed to get login users")
			return nil, err
		}
	case UserTypeOfCreated:
		if opts.KindSelector == "connect" {
			users, err = us.getConnectCreatedUsers(ctx, req.Range)
			if err != nil {
				return nil, err
			}
		} else if opts.KindSelector == "ai" {
			users, err = us.getAICreatedUsers(ctx, req.Range)
			if err != nil {
				return nil, err
			}
		}
	case UserTypeOfUsed:
		if opts.KindSelector == "connect" {
			users, err = us.getConnectUsedUsers(ctx, req.Range)
			if err != nil {
				return nil, err
			}
		} else if opts.KindSelector == "ai" {
			users, err = us.getAIUsedUsers(ctx, req.Range)
			if err != nil {
				return nil, err
			}
		}
	case UserTypeOfConnectionTemplateCreated:
		users, err = us.getConnectionTemplateCreatedUsers(ctx, pg.Range)
		if err != nil {
			return nil, err
		}
	default:
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}

	query := addFilter(ctx, req.FilterStack)
	query["oidc_id"] = bson.M{
		"$in": users,
	}

	pg.Total = int64(len(users))
	if pg.Direction == "asc" {
		sort = bson.M{pg.SortBy: 1}
	} else if pg.Direction == "desc" {
		sort = bson.M{pg.SortBy: -1}
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.userStatColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				P: pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, user)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listPremiumUsers(ctx context.Context, pg api.Page, filters api.FilterStack, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	query := addFilter(ctx, filters)
	if opts.KindSelector == "ai" {
		query["class.ai.premium"] = true
	} else if opts.KindSelector == "connect" {
		query["class.connect.premium"] = true
	}
	cnt, err := us.userStatColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	if cnt == 0 {
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}
	if cnt <= skip {
		return nil, api.ErrPageArgumentsTooLarge
	}

	pg.Total = cnt
	if pg.Direction == "asc" {
		sort = bson.M{pg.SortBy: 1}
	} else if pg.Direction == "desc" {
		sort = bson.M{pg.SortBy: -1}
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.userStatColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				P: pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		ctype := ""
		if opts.KindSelector == "ai" {
			ctype = user.Class.AI.Plan.Type
		} else if opts.KindSelector == "connect" {
			ctype = user.Class.Connect.Plan.Type
		}
		if opts.KindSelector == "ai" {
			credits, err := us.getUserCredits(ctx, user.OID, ctype)
			if err != nil {
				return nil, err
			}
			user.Credits = credits
		}
		list = append(list, user)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listHighKnownledgeBaseUsers(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  int64  = 0
		limit int64  = 50
		sort  bson.M = bson.M{"usages.ai.knowledge_base": -1}
	)

	if pg.PageSize != 0 {
		limit = pg.PageSize
	}

	if opts.KindSelector != "ai" {
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}

	query := bson.M{
		"created_at": bson.M{
			"$gte": time.Now().Add(-1 * TimeDurationOfWeek),
			// "$lte": now,
		},
		"class.ai.premium":         false,
		"bills.ai.total":           bson.M{"$ne": 0},
		"usages.ai.knowledge_base": bson.M{"$gte": 1},
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.userStatColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				List: []interface{}{},
				P:    pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, user)
	}

	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listNoKnownledgeBaseUsers(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  int64  = 0
		limit int64  = 50
		sort  bson.M = bson.M{"bills.ai.last_week": -1}
	)

	if pg.PageSize != 0 {
		limit = pg.PageSize
	}

	if opts.KindSelector != "ai" {
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}

	// users, usage, err := us.getLastWeekUsageUserList(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	query := bson.M{
		"created_at": bson.M{
			"$gte": time.Now().Add(-1 * TimeDurationOfWeek),
			// "$lte": end,
		},
		"bills.ai.total":           bson.M{"$ne": 0},
		"usages.ai.knowledge_base": 0,
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.userStatColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				List: []interface{}{},
				P:    pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, user)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listCohortUsers(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	query := bson.M{}
	opt := options.FindOptions{
		Sort: bson.M{"week.number": 1},
	}
	cursor, err := us.cohortColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				List: []interface{}{},
				P:    pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		cohort := &models.WeeklyCohortAnalysis{}
		if err = cursor.Decode(cohort); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, cohort)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listDailyUserNumber(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	query := bson.M{
		"date": bson.M{
			"$gte": GetStartAt(ctx, pg.Range),
			// "$lte": end,
		},
		"tag": pg.Tag,
	}
	opt := options.FindOptions{
		Sort: bson.M{"date": 1},
	}
	cursor, err := us.dailyStatColl.Find(ctx, query, &opt)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return &api.ListResult{
				List: []interface{}{},
				P:    pg,
			}, nil
		}
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()
	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		daily := &models.DailyStatsOfUserNumber{}
		if err = cursor.Decode(daily); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, daily)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) Get(ctx context.Context, oid string, opts *api.GetOptions) (*models.UserDetail, error) {
	if opts.KindSelector == "" {
		ai, err := us.getAIDetail(ctx, oid)
		if err != nil {
			return nil, err
		}
		connect, err := us.getConnectDetail(ctx, oid)
		if err != nil {
			return nil, err
		}
		return &models.UserDetail{
			AI:      ai,
			Connect: connect,
		}, nil
	} else if opts.KindSelector == "ai" {
		ai, err := us.getAIDetail(ctx, oid)
		if err != nil {
			return nil, err
		}
		return &models.UserDetail{
			AI: ai,
		}, nil
	} else if opts.KindSelector == "connect" {
		connect, err := us.getConnectDetail(ctx, oid)
		if err != nil {
			return nil, err
		}
		return &models.UserDetail{
			Connect: connect,
		}, nil
	}
	return nil, api.ErrUnsupportedKind.WithMessage(fmt.Sprintf("unsupported kind: %s", opts.KindSelector))
}

func addFilter(ctx context.Context, fs api.FilterStack) bson.M {
	if fs.Filters == nil {
		return bson.M{}
	}
	if len(fs.Filters) == 0 {
		return bson.M{}
	}
	var (
		at  time.Time
		err error
	)
	results := make([]bson.M, 0)
	for _, column := range fs.Filters {
		if column.Operator == "isBefore" || column.Operator == "isAfter" {
			at, err = time.Parse("2006-01-02T15:04:05.000", column.Value)
			if err != nil {
				continue
			}
		}
		switch column.Operator {
		case "includes":
			results = append(results, bson.M{column.ColumnID: bson.M{"$regex": column.Value}})
		case "doesNotInclude":
			results = append(results, bson.M{column.ColumnID: bson.M{"$not": bson.M{"$regex": column.Value}}})
		case "is":
			results = append(results, bson.M{column.ColumnID: bson.M{"$eq": column.Value}})
		case "isNot":
			results = append(results, bson.M{column.ColumnID: bson.M{"$ne": column.Value}})
		case "isEmpty":
			results = append(results, bson.M{column.ColumnID: bson.M{"$exists": false}})
		case "isNotEmpty":
			results = append(results, bson.M{column.ColumnID: bson.M{"$exists": true}})
		case "isBefore":
			results = append(results, bson.M{column.ColumnID: bson.M{"$lte": at}})
		case "isAfter":
			results = append(results, bson.M{column.ColumnID: bson.M{"$gte": at}})
		}
	}
	query := bson.M{}
	if fs.Operator == "or" {
		query["$or"] = results
	} else {
		query["$and"] = results
	}
	return query
}

func (us *UserService) getUserCredits(ctx context.Context, oid string, ctype string) (*models.Credits, error) {
	now := time.Now()
	query := bson.M{
		"user_id": oid,
		"kind":    ctype,
		"type":    "monthly",
		"period_of_validity.start": bson.M{
			"$lte": now,
		},
		"period_of_validity.end": bson.M{
			"$gte": now,
		},
	}
	cursor, err := us.creditColl.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	credits := make([]*cloud.UserCredits, 0)
	for cursor.Next(ctx) {
		credit := &cloud.UserCredits{}
		if err = cursor.Decode(credit); err != nil {
			return nil, err
		}
		credits = append(credits, credit)
	}
	if len(credits) == 0 {
		return &models.Credits{}, nil
	}
	result := &models.Credits{
		Used:     credits[0].Used,
		Total:    uint64(credits[0].Total),
		UsageStr: fmt.Sprintf("%d/%d", credits[0].Used, credits[0].Total),
	}
	return result, nil
}

// func (us *UserService) getWeeklyRetentions(ctx context.Context, week *models.Week, kind string) (map[string]*models.WeeklyRetention, uint64, error) {
// 	totalUsers := uint64(0)
// 	weeklyActiveUsers := make(map[string]uint64)
// 	retentions := make(map[string]*models.WeeklyRetention)
// 	query := bson.M{
// 		"created_at": bson.M{
// 			"$gte": week.Start,
// 			"$lte": week.End,
// 		},
// 	}
// 	cursor, err := us.userStatColl.Find(ctx, query)
// 	if err != nil {
// 		return retentions, 0, err
// 	}
// 	defer func() {
// 		_ = cursor.Close(ctx)
// 	}()
// 	for cursor.Next(ctx) {
// 		user := &models.User{}
// 		if err = cursor.Decode(user); err != nil {
// 			return nil, 0, err
// 		}
// 		totalUsers += 1
// 		if kind != "ai" {
// 			for weekNum, retention := range user.Cohort.AI {
// 				if _, ok := retentions[weekNum]; ok {
// 					retentions[weekNum].Usage += retention.Usage
// 				} else {
// 					retentions[weekNum] = &models.WeeklyRetention{
// 						Week:  retention.Week,
// 						Usage: retention.Usage,
// 					}
// 				}
// 				if retention.Active {
// 					weeklyActiveUsers[weekNum] += 1
// 				}
// 			}
// 		} else if kind != "connect" {
// 			for weekNum, retention := range user.Cohort.Connect {
// 				if _, ok := retentions[weekNum]; ok {
// 					retentions[weekNum].Usage += retention.Usage
// 				} else {
// 					retentions[weekNum] = &models.WeeklyRetention{
// 						Week:  retention.Week,
// 						Usage: retention.Usage,
// 					}
// 				}
// 				if retention.Active {
// 					weeklyActiveUsers[weekNum] += 1
// 				}
// 			}
// 		}
// 		// log.Info(ctx).Uint64("cnt", cnt).Any("user", user).Msg("success to get user")
// 	}
// 	for weekNum, activeNum := range weeklyActiveUsers {
// 		ratio := math.Round(float64(activeNum)/float64(totalUsers)*10000) / 100
// 		retentions[weekNum].Ratio = fmt.Sprintf("%0.2f%%", ratio)
// 	}
// 	return retentions, totalUsers, nil
// }

func (us *UserService) getLastWeekCreatedKnowledgeBaseUserList(ctx context.Context) ([]string, error) {
	query := bson.M{
		"created_at": bson.M{
			"$gte": time.Now().Add(-1 * TimeDurationOfWeek),
			// "$lte": end,
		},
	}
	cursor, err := us.aiKnowledgeBaseColl.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	users := make([]string, 0)
	for cursor.Next(ctx) {
		knowledge := &cloud.KnowledgeBase{}
		if err = cursor.Decode(knowledge); err != nil {
			return nil, err
		}
		users = append(users, knowledge.CreatedBy)
	}
	return users, nil
}

func (us *UserService) getLastWeekUsageUserList(ctx context.Context) ([]string, map[string]uint64, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.M{
				"collected_at": bson.M{
					"$gte": time.Now().Add(-1 * TimeDurationOfWeek),
					// "$lte": end,
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
		{
			{"$sort", bson.D{
				{"usage", -1},
			}},
		},
	}
	type usageGroup struct {
		User  string `bson:"_id"`
		Usage uint64 `bson:"usage"`
	}
	cursor, err := us.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, nil, err
	}
	defer cursor.Close(ctx)
	userList := make([]string, 0)
	userMap := make(map[string]uint64, 0)
	for cursor.Next(ctx) {
		var usageGroup usageGroup
		if err = cursor.Decode(&usageGroup); err != nil {
			return nil, nil, err
		}
		userList = append(userList, usageGroup.User)
		userMap[usageGroup.User] = usageGroup.Usage
	}
	return userList, userMap, nil
}

func (us *UserService) getLoginUsers(ctx context.Context, rg api.Range) ([]string, error) {
	layout := "2006-01-02"
	now := time.Now()
	if rg.Start == "" || rg.End == "" {
		rg.Start = now.AddDate(0, 0, -1).Format(layout)
		rg.End = now.Format(layout)
	}
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"time", bson.M{
					"$gte": rg.Start,
					"$lte": rg.End,
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
	cursor, err := us.actionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	users := make([]string, 0)
	type userGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	for cursor.Next(ctx) {
		var ug userGroup
		if err = cursor.Decode(&ug); err != nil {
			return nil, err
		}
		users = append(users, ug.User)
	}
	return users, nil
}

func (us *UserService) getConnectCreatedUsers(ctx context.Context, rg api.Range) ([]string, error) {
	start, end, err := us.getRangeTime(ctx, rg)
	if err != nil {
		return nil, err
	}
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
		{
			{"$sort", bson.D{
				{"count", -1},
			}},
		},
	}
	cursor, err := us.connectionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	type userGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	users := make([]string, 0)
	for cursor.Next(ctx) {
		var ug userGroup
		if err = cursor.Decode(&ug); err != nil {
			return nil, err
		}
		users = append(users, ug.User)
	}
	return users, nil
}

func (us *UserService) getAICreatedUsers(ctx context.Context, rg api.Range) ([]string, error) {
	start, end, err := us.getRangeTime(ctx, rg)
	if err != nil {
		return nil, err
	}
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
	cursor, err := us.appColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	type userGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	users := make([]string, 0)
	for cursor.Next(ctx) {
		var ug userGroup
		if err = cursor.Decode(&ug); err != nil {
			return nil, err
		}
		users = append(users, ug.User)
	}
	return users, nil
}

func (us *UserService) getConnectUsedUsers(ctx context.Context, rg api.Range) ([]string, error) {
	start, end, err := us.getRangeTime(ctx, rg)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"collected_at", bson.M{
					"$gt":  start,
					"$lte": end,
				}},
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
	cursor, err := us.billColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	type userGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	users := make([]string, 0)
	for cursor.Next(ctx) {
		var ug userGroup
		if err = cursor.Decode(&ug); err != nil {
			return nil, err
		}
		users = append(users, ug.User)
	}
	return users, nil
}

func (us *UserService) getAIUsedUsers(ctx context.Context, rg api.Range) ([]string, error) {
	start, end, err := us.getRangeTime(ctx, rg)
	if err != nil {
		return nil, err
	}
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
	cursor, err := us.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	type userGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	users := make([]string, 0)
	for cursor.Next(ctx) {
		var ug userGroup
		if err = cursor.Decode(&ug); err != nil {
			return nil, err
		}
		users = append(users, ug.User)
	}
	return users, nil
}

func (us *UserService) getConnectionTemplateCreatedUsers(ctx context.Context, rg string) ([]string, error) {
	date, err := time.Parse("2006-01-02", rg)
	if err != nil {
		return nil, err
	}
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"template_id", bson.M{
					"$exists": true,
				}},
				{"created_at", bson.M{
					"$gte": date,
					"$lte": date.AddDate(0, 0, 1),
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$created_by"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
		{
			{"$sort", bson.D{
				{"count", -1},
			}},
		},
	}
	cursor, err := us.connectionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	type userGroup struct {
		User  string `bson:"_id"`
		Count int64  `bson:"count"`
	}
	users := make([]string, 0)
	for cursor.Next(ctx) {
		var ug userGroup
		if err = cursor.Decode(&ug); err != nil {
			return nil, err
		}
		users = append(users, ug.User)
	}
	return users, nil
}

func GetStartAt(ctx context.Context, rg string) time.Time {
	now := time.Now()
	switch rg {
	case "Month":
		startAt := now.AddDate(0, -1, 0)
		return time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 0, 0, 0, 0, time.UTC)
	case "Three Months":
		startAt := now.AddDate(0, -3, 0)
		return time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 0, 0, 0, 0, time.UTC)
	case "Six Months":
		startAt := now.AddDate(0, -6, 0)
		return time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 0, 0, 0, 0, time.UTC)
	case "Year":
		startAt := now.AddDate(-1, 0, 0)
		if startAt.Before(StartAt) {
			return StartAt
		}
		return time.Date(startAt.Year(), startAt.Month(), startAt.Day(), 0, 0, 0, 0, time.UTC)
	default:
		return StartAt
	}
}

func (us *UserService) getRangeTime(ctx context.Context, rg api.Range) (time.Time, time.Time, error) {
	now := time.Now()
	layout := "2006-01-02"
	parseTimeFromString := func(t string) (time.Time, error) {
		if strings.Contains(t, "T") {
			t = strings.Split(t, "T")[0]
		}
		return time.Parse(layout, t)
	}

	if rg.Start == "" && rg.End == "" {
		return now.AddDate(0, 0, -1), now, nil
	} else if rg.Start == "" {
		end, err := parseTimeFromString(rg.End)
		if err != nil {
			return now, now, err
		}
		return end.AddDate(0, 0, -1), end, nil
	} else if rg.End == "" {
		start, err := parseTimeFromString(rg.Start)
		if err != nil {
			return now, now, err
		}
		return start, start.AddDate(0, 0, 1), nil
	}
	start, err := parseTimeFromString(rg.Start)
	if err != nil {
		return now, now, err
	}
	if rg.Start == rg.End {
		return start, start.AddDate(0, 0, 1), nil
	}
	end, err := parseTimeFromString(rg.End)
	if err != nil {
		return now, now, err
	}
	return start, end, nil
}
