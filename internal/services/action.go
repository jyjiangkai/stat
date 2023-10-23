package services

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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
	DatabaseOfUserAnalytics        = "vanus_user_analytics"
	ActionType                     = "action_type"
	ActionTypeOfChat               = "chat"
	ActionTypeOfRedirectChangePlan = "redirect_change_plan"
)

type ActionService struct {
	cli        *mongo.Client
	appColl    *mongo.Collection
	statColl   *mongo.Collection
	actionColl *mongo.Collection
	trackColl  *mongo.Collection
	appCache   sync.Map
	closeC     chan struct{}
}

func NewActionService(cli *mongo.Client) *ActionService {
	return &ActionService{
		cli:        cli,
		appColl:    cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		statColl:   cli.Database(db.GetDatabaseName()).Collection("stats"),
		actionColl: cli.Database(DatabaseOfUserAnalytics).Collection("user_actions"),
		trackColl:  cli.Database(DatabaseOfUserAnalytics).Collection("user_tracks"),
		closeC:     make(chan struct{}),
	}
}

func (as *ActionService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		defer log.Warn(ctx).Err(nil).Msg("update user action time routine exit")
		for {
			select {
			case <-as.closeC:
				log.Info(ctx).Msg("action service stopped.")
				return
			case <-ticker.C:
				err := as.UpdateTime(ctx)
				if err != nil {
					log.Error(ctx).Err(err).Msgf("failed to update user action time at %+v", time.Now())
				}
				now := time.Now()
				if now.Weekday() == time.Monday && now.Hour() == 0 && now.Minute() == 0 {
					as.weeklyViewPriceUserTracking(ctx, now)
				}
			}
		}
	}()
	return nil
}

func (as *ActionService) weeklyViewPriceUserTracking(ctx context.Context, now time.Time) error {
	log.Info(ctx).Msgf("start stat weekly view price user at: %+v\n", now)
	users, counts, err := as.getViewPriceUsers(ctx, now)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to get view price users")
		return err
	}

	query := bson.M{
		"oidc_id": bson.M{
			"$in": users,
		},
		"email": bson.M{
			"$not": bson.M{
				"$regex": "linkall.com|vanus.ai",
			},
		},
		"class.ai.premium":         false,
		"class.connect.premium":    false,
		"bills.ai.total":           bson.M{"$ne": 0},
		"usages.ai.knowledge_base": bson.M{"$gte": 2},
	}
	cursor, err := as.statColl.Find(ctx, query)
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	for cursor.Next(ctx) {
		user := &models.User{}
		if err = cursor.Decode(user); err != nil {
			return db.HandleDBError(err)
		}
		track := &models.Track{
			User:  user.OID,
			Tag:   ActionTypeOfRedirectChangePlan,
			Count: counts[user.OID],
			Time:  time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC),
		}
		_, err := as.trackColl.InsertOne(ctx, track)
		if err != nil {
			log.Error(ctx).Err(err).Any("track", track).Msg("failed to insert user track")
			return db.HandleDBError(err)
		}
		if mailchimp.ValidateEmail(user.Email) {
			tags := []string{ActionTypeOfRedirectChangePlan}
			err := mailchimp.AddMember(ctx, user.Email, tags)
			if err != nil {
				log.Error(ctx).Str("email", user.Email).Msg("failed to add member to mailchimp")
			}
		}
	}
	log.Info(ctx).Msgf("finish stat weekly view price user at: %+v\n", time.Now())
	return nil
}

func (as *ActionService) getViewPriceUsers(ctx context.Context, now time.Time) ([]string, map[string]uint64, error) {
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
	cursor, err := as.actionColl.Aggregate(ctx, pipeline)
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

func (as *ActionService) Stop() error {
	return nil
}

func (as *ActionService) List(ctx context.Context, pg api.Page, filter api.Filter, opts *api.ListOptions) (*api.ListResult, error) {
	log.Info(ctx).Any("page", pg).Any("filter", filter).Any("opts", opts).Msg("action service list api")
	switch opts.TypeSelector {
	case ActionType:
		return as.listSpecifiedActionTypeUsers(ctx, pg, filter, opts)
	default:
		return as.list(ctx, pg, filter, opts)
	}
}

func (as *ActionService) list(ctx context.Context, pg api.Page, filter api.Filter, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	query := addActionFilter(ctx, filter)
	query["website"] = bson.M{"$ne": "https://ai.vanustest.com"}
	log.Info(ctx).Any("query", query).Msg("show action list api query criteria")
	cnt, err := as.actionColl.CountDocuments(ctx, query)
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
	} else {
		sort = bson.M{pg.SortBy: -1}
	}

	opt := options.FindOptions{
		Limit: &limit,
		Skip:  &skip,
		Sort:  sort,
	}
	cursor, err := as.actionColl.Find(ctx, query, &opt)
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
		action := &models.Action{}
		if err = cursor.Decode(action); err != nil {
			return nil, db.HandleDBError(err)
		}
		if action.Payload.AppID != "" {
			if app, ok := as.appCache.Load(action.Payload.AppID); ok {
				action.App = app.(*models.ActionApp)
			} else {
				id, _ := primitive.ObjectIDFromHex(action.Payload.AppID)
				result := as.appColl.FindOne(ctx, bson.M{"_id": id})
				if result.Err() != nil {
					return nil, db.HandleDBError(result.Err())
				}
				app := &cloud.App{}
				if err := result.Decode(app); err != nil {
					return nil, db.HandleDBError(err)
				}
				actionApp := &models.ActionApp{
					Name:     app.Name,
					Type:     app.Type,
					Model:    app.Model,
					Status:   string(app.Status),
					Greeting: app.Greeting,
					Prompt:   app.Prompt,
				}
				action.App = actionApp
				as.appCache.Store(action.Payload.AppID, actionApp)
			}
		} else {
			action.App = models.NewActionApp()
		}
		list = append(list, action)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (as *ActionService) listSpecifiedActionTypeUsers(ctx context.Context, pg api.Page, filter api.Filter, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	query, atype := genQueryFromActionTypeFilter(ctx, filter)
	query["website"] = bson.M{"$ne": "https://ai.vanustest.com"}
	cnt, err := as.actionColl.CountDocuments(ctx, query)
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
	pg.SortBy = "count"
	if pg.Direction == "asc" {
		sort = bson.M{pg.SortBy: 1}
	} else if pg.Direction == "desc" {
		sort = bson.M{pg.SortBy: -1}
	} else {
		sort = bson.M{pg.SortBy: -1}
	}

	pipeline := mongo.Pipeline{
		{
			{"$match", query},
		},
		{
			{"$group", bson.D{
				{"_id", "$usersub"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
		{
			{"$sort", sort},
		},
		// {
		// 	{"$sort", bson.D{
		// 		{"count", -1},
		// 	}},
		// },
	}
	type countGroup struct {
		User   string `bson:"_id"`
		Action string `bson:"action"`
		Count  uint64 `bson:"count"`
	}
	cursor, err := as.actionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)
	list := make([]interface{}, 0)
	for cursor.Next(ctx) {
		var cg countGroup
		if err = cursor.Decode(&cg); err != nil {
			return nil, err
		}
		cg.Action = atype
		list = append(list, cg)
	}
	return &api.ListResult{
		List: list[skip : skip+limit],
		P:    pg,
	}, nil
}

func (as *ActionService) Get(ctx context.Context, oid string, opts *api.GetOptions) (*models.UserDetail, error) {
	return nil, nil
}

func (as *ActionService) UpdateTime(ctx context.Context) error {
	query := bson.M{}
	query["time"] = bson.M{"$regex": "^16.*"}
	cursor, err := as.actionColl.Find(ctx, query)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil
		}
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	type ActionOfTimeStamp struct {
		ID   primitive.ObjectID `json:"id" bson:"_id"`
		Time string             `bson:"time"`
	}
	for cursor.Next(ctx) {
		action := &ActionOfTimeStamp{}
		if err = cursor.Decode(action); err != nil {
			return db.HandleDBError(err)
		}
		realTime, err := toRealTime(ctx, action.Time)
		if err != nil {
			log.Error(ctx).Err(err).Msgf("failed to parse timestamp str: %s", action.Time)
		}
		_, err = as.actionColl.UpdateOne(ctx,
			bson.M{"_id": action.ID},
			bson.M{
				"$set": bson.M{
					"time": realTime,
				}})
		if err != nil {
			log.Error(ctx).Err(err).Str("id", action.ID.Hex()).Msg("failed to update time")
		}
	}
	return nil
}

func toRealTime(ctx context.Context, timestampStr string) (string, error) {
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return "", err
	}
	return time.Unix(0, timestamp).UTC().Format(time.RFC3339), nil
}

func genQueryFromActionTypeFilter(ctx context.Context, filter api.Filter) (bson.M, string) {
	if filter.Columns == nil {
		return bson.M{"action": bson.M{"$eq": ActionTypeOfChat}}, ActionTypeOfChat
	}
	if len(filter.Columns) == 0 {
		return bson.M{"action": bson.M{"$eq": ActionTypeOfChat}}, ActionTypeOfChat
	}
	if len(filter.Columns) > 1 {
		return bson.M{"action": bson.M{"$eq": ActionTypeOfChat}}, ActionTypeOfChat
	}
	var actionType string
	results := make([]bson.M, 0)
	for _, column := range filter.Columns {
		if !strings.EqualFold(column.ColumnID, "action") {
			continue
		}
		key := strings.ToLower(column.ColumnID)
		actionType = column.Operator + " " + column.Value
		switch column.Operator {
		case "includes":
			results = append(results, bson.M{key: bson.M{"$regex": column.Value}})
		case "doesNotInclude":
			results = append(results, bson.M{key: bson.M{"$not": bson.M{"$eq": column.Value}}})
		case "is":
			results = append(results, bson.M{key: bson.M{"$eq": column.Value}})
		case "isNot":
			results = append(results, bson.M{key: bson.M{"$ne": column.Value}})
		case "isEmpty":
			results = append(results, bson.M{key: bson.M{"$exists": false}})
		case "isNotEmpty":
			results = append(results, bson.M{key: bson.M{"$exists": true}})
		}
	}
	query := bson.M{}
	if filter.Operator == "or" {
		query["$or"] = results
	} else {
		query["$and"] = results
	}
	return query, actionType
}

func addActionFilter(ctx context.Context, filter api.Filter) bson.M {
	if filter.Columns == nil {
		return bson.M{}
	}
	if len(filter.Columns) == 0 {
		return bson.M{}
	}
	results := make([]bson.M, 0)
	for _, column := range filter.Columns {
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
			results = append(results, bson.M{column.ColumnID: bson.M{"$lte": column.Value}})
		case "isAfter":
			results = append(results, bson.M{column.ColumnID: bson.M{"$gte": column.Value}})
		}
	}
	query := bson.M{}
	if filter.Operator == "or" {
		query["$or"] = results
	} else {
		query["$and"] = results
	}
	return query
}
