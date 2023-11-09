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
	DatabaseOfUserAnalytics            = "vanus_user_analytics"
	ConnectionTemplateCreatedNumber    = "connection_template_created_number"
	ConnectionTemplateCreated          = "connection_template_created"
	ActionType                         = "action_type"
	DailyActionNumber                  = "daily_action_number"
	ActionTypeOfChat                   = "chat"
	ActionTypeOfRedirectChangePlan     = "redirect_change_plan"
	ActionTypeOfSwitchSidebarKnowledge = "switch_sidebar_knowledge"
)

type ActionService struct {
	cli            *mongo.Client
	appColl        *mongo.Collection
	connectionColl *mongo.Collection
	chatColl       *mongo.Collection
	statColl       *mongo.Collection
	dailyStatColl  *mongo.Collection
	actionColl     *mongo.Collection
	trackColl      *mongo.Collection
	appCache       sync.Map
	closeC         chan struct{}
}

func NewActionService(cli *mongo.Client) *ActionService {
	return &ActionService{
		cli:            cli,
		appColl:        cli.Database(db.GetDatabaseName()).Collection("ai_app"),
		connectionColl: cli.Database(db.GetDatabaseName()).Collection("connections"),
		chatColl:       cli.Database(db.GetDatabaseName()).Collection("ai_chat_history"),
		statColl:       cli.Database(DatabaseOfUserStatistics).Collection("user_stats"),
		dailyStatColl:  cli.Database(DatabaseOfUserStatistics).Collection("daily_stats"),
		actionColl:     cli.Database(DatabaseOfUserAnalytics).Collection("user_actions"),
		trackColl:      cli.Database(DatabaseOfUserAnalytics).Collection("user_tracks"),
		closeC:         make(chan struct{}),
	}
}

func (as *ActionService) Start() error {
	ctx := context.Background()
	go func() {
		ticker := time.NewTicker(time.Hour)
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
				if now.Weekday() == time.Monday && now.Hour() == 2 {
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
		"usages.ai.knowledge_base": bson.M{"$gte": 1},
	}
	cursor, err := as.statColl.Find(ctx, query)
	if err != nil {
		return err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	cnt := 0
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
		cnt += 1
	}
	log.Info(ctx).Int("cnt", cnt).Msgf("finish stat weekly view price user at: %+v\n", time.Now())
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

func (as *ActionService) List(ctx context.Context, pg api.Page, filters api.FilterStack, opts *api.ListOptions) (*api.ListResult, error) {
	log.Info(ctx).Any("page", pg).Any("filters", filters).Any("opts", opts).Msg("action service list api")
	switch opts.TypeSelector {
	case ConnectionTemplateCreatedNumber:
		return as.listConnectionTemplateCreatedNumber(ctx, pg, opts)
	case ConnectionTemplateCreated:
		return as.listConnectionTemplateCreated(ctx, pg, filters, opts)
	case ActionType:
		return as.listSpecifiedActionTypeUsers(ctx, pg, filters, opts)
	case DailyActionNumber:
		return as.listDailyActionNumber(ctx, pg, opts)
	default:
		return as.list(ctx, pg, filters, opts)
	}
}

func (as *ActionService) list(ctx context.Context, pg api.Page, filters api.FilterStack, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	query := addActionFilter(ctx, filters)
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

func (as *ActionService) listConnectionTemplateCreatedNumber(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"template_id", bson.M{"$exists": true}},
			}},
		},
		{
			{"$project", bson.D{
				{"date", bson.D{
					{"$dateToString", bson.M{"format": "%Y-%m-%d", "date": "$created_at"}},
				}},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$date"},
				{"count", bson.M{"$sum": 1}},
			}},
		},
		{
			{"$sort", bson.D{
				{"_id", 1},
			}},
		},
	}
	if pg.Range != "" {
		pipeline[0] = bson.D{
			{"$match", bson.D{
				{"template_id", pg.Range},
			}},
		}
	}
	type countGroup struct {
		Date  string `bson:"_id"`
		Count uint64 `bson:"count"`
	}
	cursor, err := as.connectionColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Msg("no documents")
		}
		log.Error(ctx).Err(err).Msg("aggregate error")
		return nil, err
	}
	defer cursor.Close(ctx)

	layout := "2006-01-02"
	results := make(map[string]countGroup, 0)
	for cursor.Next(ctx) {
		var cg countGroup
		if err = cursor.Decode(&cg); err != nil {
			return nil, err
		}
		results[cg.Date] = cg
	}
	list := make([]interface{}, 0)
	now := time.Now()
	date := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
	for {
		timeStr := date.Format(layout)
		if cg, ok := results[timeStr]; ok {
			list = append(list, countGroup{
				Date:  timeStr,
				Count: cg.Count,
			})
		} else {
			list = append(list, countGroup{
				Date:  timeStr,
				Count: 0,
			})
		}
		if timeStr == now.Format(layout) {
			break
		}
		date = date.AddDate(0, 0, 1)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (as *ActionService) listConnectionTemplateCreated(ctx context.Context, pg api.Page, filters api.FilterStack, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	if pg.Range == "" || pg.Range == "null" {
		return &api.ListResult{
			List: []interface{}{},
			P:    pg,
		}, nil
	}

	date, err := time.Parse("2006-01-02", pg.Range)
	if err != nil {
		return nil, err
	}
	query := addActionFilter(ctx, filters)
	query["created_at"] = bson.M{
		"$gte": date,
		"$lte": date.AddDate(0, 0, 1),
	}
	if pg.Tag == "" || pg.Tag == "all" {
		query["template_id"] = bson.M{"$exists": true}
	} else {
		query["template_id"] = pg.Range
	}
	log.Info(ctx).Any("query", query).Msg("show connection template created list api query criteria")
	cnt, err := as.connectionColl.CountDocuments(ctx, query)
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
	cursor, err := as.connectionColl.Find(ctx, query, &opt)
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
		cn := &models.Connection{}
		if err = cursor.Decode(cn); err != nil {
			return nil, db.HandleDBError(err)
		}
		obtainConnectionTemplateNameFromID(cn)
		list = append(list, cn)
	}
	return &api.ListResult{
		List: list,
		P:    pg,
	}, nil
}

func (as *ActionService) listSpecifiedActionTypeUsers(ctx context.Context, pg api.Page, filters api.FilterStack, opts *api.ListOptions) (*api.ListResult, error) {
	var (
		skip  = pg.PageNumber * pg.PageSize
		limit = pg.PageSize
		sort  bson.M
	)

	if skip < 0 {
		skip = 0
	}

	query, atype := genQueryFromActionTypeFilter(ctx, filters)
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

func (as *ActionService) listDailyActionNumber(ctx context.Context, pg api.Page, opts *api.ListOptions) (*api.ListResult, error) {
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
	cursor, err := as.dailyStatColl.Find(ctx, query, &opt)
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
		daily := &models.Daily{}
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

func (as *ActionService) Get(ctx context.Context, oid string, opts *api.GetOptions) (*models.UserDetail, error) {
	return nil, nil
}

func (as *ActionService) UpdateTime(ctx context.Context) error {
	err := as.UpdateActionTime(ctx)
	if err != nil {
		return err
	}
	err = as.UpdateChatHistoryTime(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (as *ActionService) UpdateActionTime(ctx context.Context) error {
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

func (as *ActionService) UpdateChatHistoryTime(ctx context.Context) error {
	query := bson.M{}
	query["time"] = bson.M{"$gte": 1690000000000}
	cursor, err := as.chatColl.Find(ctx, query)
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
		Time int64              `bson:"time"`
	}
	for cursor.Next(ctx) {
		action := &ActionOfTimeStamp{}
		if err = cursor.Decode(action); err != nil {
			return db.HandleDBError(err)
		}
		_, err = as.chatColl.UpdateOne(ctx,
			bson.M{"_id": action.ID},
			bson.M{
				"$set": bson.M{
					"time": toFormatTime(ctx, action.Time),
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

func toFormatTime(ctx context.Context, timestampInt int64) time.Time {
	return time.Unix(0, timestampInt*int64(time.Millisecond)).UTC()
}

func genQueryFromActionTypeFilter(ctx context.Context, filters api.FilterStack) (bson.M, string) {
	if filters.Filters == nil {
		return bson.M{"action": bson.M{"$eq": ActionTypeOfChat}}, ActionTypeOfChat
	}
	if len(filters.Filters) == 0 {
		return bson.M{"action": bson.M{"$eq": ActionTypeOfChat}}, ActionTypeOfChat
	}
	if len(filters.Filters) > 1 {
		return bson.M{"action": bson.M{"$eq": ActionTypeOfChat}}, ActionTypeOfChat
	}
	var actionType string
	results := make([]bson.M, 0)
	for _, column := range filters.Filters {
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
	if filters.Operator == "or" {
		query["$or"] = results
	} else {
		query["$and"] = results
	}
	return query, actionType
}

func addActionFilter(ctx context.Context, filters api.FilterStack) bson.M {
	if filters.Filters == nil {
		return bson.M{}
	}
	if len(filters.Filters) == 0 {
		return bson.M{}
	}
	results := make([]bson.M, 0)
	for _, column := range filters.Filters {
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
	if filters.Operator == "or" {
		query["$or"] = results
	} else {
		query["$and"] = results
	}
	return query
}

func obtainConnectionTemplateNameFromID(cn *models.Connection) {
	switch cn.Template {
	case "20230306_1":
		cn.SourceType = "Github"
		cn.SinkType = "Feishu"
	case "20230329_0":
		cn.SourceType = "ChatGPT"
		cn.SinkType = "Feishu"
	case "20230425_4":
		cn.SourceType = "Github"
		cn.SinkType = "Snowflake"
	case "20230525_1":
		cn.SourceType = "Whatsapp"
		cn.SinkType = "Whatsapp"
	case "20231023_2":
		cn.SourceType = "Shopify"
		cn.SinkType = "Google Sheets"
	case "20231023_4":
		cn.SourceType = "Shopify"
		cn.SinkType = "Google Sheets"
	case "20231023_5":
		cn.SourceType = "Shopify"
		cn.SinkType = "Outlook"
	case "20231023_6":
		cn.SourceType = "Shopify"
		cn.SinkType = "Outlook"
	case "20231023_7":
		cn.SourceType = "Shopify"
		cn.SinkType = "Slack"
	case "20231023_1":
		cn.SourceType = "Shopify"
		cn.SinkType = "Slack"
	default:
		cn.SourceType = "Unknown"
		cn.SinkType = "Unknown"
	}
}
