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
	"github.com/jyjiangkai/stat/mailchimp"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
)

const (
	UserTypeOfIntention = "intention"
	UserTypeOfMarketing = "marketing"
)

var (
	TimeDurationOfWeek  time.Duration = 7 * 24 * time.Hour
	TimeDurationOfMonth time.Duration = 30 * 24 * time.Hour
)

type UserService struct {
	cli                 *mongo.Client
	appColl             *mongo.Collection
	billColl            *mongo.Collection
	aiBillColl          *mongo.Collection
	promptColl          *mongo.Collection
	uploadColl          *mongo.Collection
	aiKnowledgeBaseColl *mongo.Collection
	connectorColl       *mongo.Collection
	connectionColl      *mongo.Collection
	statColl            *mongo.Collection
}

func NewUserService(cli *mongo.Client) *UserService {
	return &UserService{
		cli:                 cli,
		appColl:             cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		billColl:            cli.Database(db.GetDatabaseName()).Collection("bills"),
		aiBillColl:          cli.Database(db.GetDatabaseName()).Collection("ai_bills"),
		promptColl:          cli.Database(db.GetDatabaseName()).Collection("ai_prompt"),
		uploadColl:          cli.Database(db.GetDatabaseName()).Collection("ai_upload"),
		aiKnowledgeBaseColl: cli.Database(db.GetDatabaseName()).Collection("ai_knowledge_bases"),
		connectorColl:       cli.Database(db.GetDatabaseName()).Collection("connectors"),
		connectionColl:      cli.Database(db.GetDatabaseName()).Collection("connections"),
		statColl:            cli.Database(db.GetDatabaseName()).Collection("stats"),
	}
}

func (us *UserService) Start() error {
	return nil
}

func (us *UserService) Stop() error {
	return nil
}

func (us *UserService) List(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	log.Info(ctx).Str("kind", opts.KindSelector).Str("type", opts.TypeSelector).Msg("list users")
	switch opts.TypeSelector {
	case UserTypeOfIntention:
		return us.listIntentionUsers(ctx, pg, opts)
	case UserTypeOfMarketing:
		return us.listMarketingUsers(ctx, pg, opts)
	default:
		return us.list(ctx, pg, opts)
	}
}

func (us *UserService) list(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	query := bson.M{}
	if opts.KindSelector == "ai" {
		query["usages.ai.app"] = bson.M{"$ne": 0}
	} else if opts.KindSelector == "connect" {
		query["usages.connect.connection"] = bson.M{"$ne": 0}
	}
	cnt, err := us.statColl.CountDocuments(ctx, query)
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
	cursor, err := us.statColl.Find(ctx, query, &opt)
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

	num := 0
	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return nil, db.HandleDBError(err)
		}
		list = append(list, user)
		num += 1
	}

	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) listIntentionUsers(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
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

	// users, err := us.getLastWeekCreatedKnowledgeBaseUserList(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	query := bson.M{
		"created_at": bson.M{
			"$gte": time.Now().Add(-1 * TimeDurationOfWeek),
			// "$lte": end,
		},
		// "oidc_id":          bson.M{"$in": users},
		"class.ai.premium":         false,
		"bills.ai.total":           bson.M{"$ne": 0},
		"usages.ai.knowledge_base": bson.M{"$ne": 0},
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.statColl.Find(ctx, query, &opt)
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

func (us *UserService) listMarketingUsers(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
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
		// "oidc_id":                  bson.M{"$in": users},
		"bills.ai.total":           bson.M{"$ne": 0},
		"usages.ai.knowledge_base": 0,
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := us.statColl.Find(ctx, query, &opt)
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
		// user.Bills.AI.Total = usage[user.OID]
		if mailchimp.ValidateEmail(user.Email) {
			err := mailchimp.Subscribe(ctx, user.Email)
			if err != nil {
				log.Error(ctx).Str("email", user.Email).Msg("failed to subscribe email to mailchimp")
			}
		}
		list = append(list, user)
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
