package services

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jyjiangkai/stat/api"
	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
)

type UserService struct {
	cli            *mongo.Client
	appColl        *mongo.Collection
	billColl       *mongo.Collection
	aiBillColl     *mongo.Collection
	promptColl     *mongo.Collection
	uploadColl     *mongo.Collection
	connectorColl  *mongo.Collection
	connectionColl *mongo.Collection
	statColl       *mongo.Collection
}

func NewUserService(cli *mongo.Client) *UserService {
	return &UserService{
		cli:            cli,
		appColl:        cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		billColl:       cli.Database(db.GetDatabaseName()).Collection("bills"),
		aiBillColl:     cli.Database(db.GetDatabaseName()).Collection("ai_bills"),
		promptColl:     cli.Database(db.GetDatabaseName()).Collection("ai_prompt"),
		uploadColl:     cli.Database(db.GetDatabaseName()).Collection("ai_upload"),
		connectorColl:  cli.Database(db.GetDatabaseName()).Collection("connectors"),
		connectionColl: cli.Database(db.GetDatabaseName()).Collection("connections"),
		statColl:       cli.Database(db.GetDatabaseName()).Collection("stats"),
	}
}

func (us *UserService) Start() error {
	return nil
}

func (us *UserService) Stop() error {
	return nil
}

func (us *UserService) List(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	log.Info(ctx).Int64("page_size", pg.PageSize).Int64("page_number", pg.PageNumber).Str("sort_by", pg.SortBy).Str("direction", pg.Direction).Msg("print params of user list")
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
	log.Info(ctx).Int64("limit", limit).Int64("skip", skip).Any("sort", sort).Msg("print find options of user list")
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
