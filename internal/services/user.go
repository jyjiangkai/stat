package services

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jyjiangkai/stat/api"
	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
)

const (
	userCollName = "users"
)

type UserService struct {
	cli            *mongo.Client
	statColl       *mongo.Collection
	connectionColl *mongo.Collection
}

func NewUserService(cli *mongo.Client) *UserService {
	return &UserService{
		cli:            cli,
		statColl:       cli.Database(db.GetDatabaseName()).Collection("statistics"),
		connectionColl: cli.Database(db.GetDatabaseName()).Collection("connections"),
	}
}

func (us *UserService) Start() error {
	return nil
}

func (us *UserService) Stop() error {
	return nil
}

func (us *UserService) List(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	log.Info(ctx).Int64("page_size", pg.PageSize).Int64("page_number", pg.PageNumber).Str("sort_by", pg.SortBy).Str("direction", pg.Direction).Msg("show page params")
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
	log.Info(ctx).Int64("limit", limit).Int64("skip", skip).Int64("total", cnt).Any("sort", sort).Msg("show find opts")
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
		log.Info(ctx).Msgf("[%d] created_at: %+v\n", num, user.CreatedAt)
		list = append(list, user)
		num += 1
	}

	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (us *UserService) Get(ctx context.Context, oid string, opts *api.GetOptions) (*models.UserDetail, error) {
	// 1. get all connection from oid
	// 2. 遍历所有connection
	// 3. 通过bills获取connection的总使用量
	// 4. 获取source和sink的类型
	// 5. 获取eventbus name ？
	//

	// query := bson.M{
	// 	"created_by": oid,
	// 	// "status": bson.M{
	// 	// 	"$ne": "deleted",
	// 	// },
	// }
	// cursor, err := us.connectionColl.Find(ctx, query)
	// if err != nil {
	// 	if err == mongo.ErrNoDocuments {
	// 		return &models.UserDetail{}, nil
	// 	}
	// 	return nil, err
	// }
	// defer func() {
	// 	_ = cursor.Close(ctx)
	// }()

	// list := make([]interface{}, 0)
	// for cursor.Next(ctx) {
	// 	c := &cloud.Connection{}
	// 	if err = cursor.Decode(c); err != nil {
	// 		return nil, db.HandleDBError(err)
	// 	}
	// 	list = append(list, c)
	// }

	return nil, nil
}

// func lessFunc(a, b interface{}) bool {
// 	// 将a和b转换为你的元素类型
// 	// 假设你的元素类型是struct，其中一个字段是需要排序的字段
// 	aVal := a.(*models.Statistic).CreatedAt
// 	bVal := b.(YourType).YourFieldToSort

// 	// 比较元素值，并返回比较结果
// 	return aVal < bVal
// }
