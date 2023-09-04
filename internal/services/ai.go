package services

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"github.com/jyjiangkai/stat/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (us *UserService) getAIDetail(ctx context.Context, oid string) (*models.UserAIDetail, error) {
	apps, err := us.getApps(ctx, oid)
	if err != nil {
		return nil, err
	}
	applications := make([]*models.App, 0)
	for idx := range apps {
		app := apps[idx]
		bills, usage, err := us.getAppBills(ctx, app.ID)
		if err != nil {
			return nil, err
		}
		newApp := &models.App{
			Base:            app.Base,
			Name:            app.Name,
			Type:            app.Type,
			Model:           app.Model,
			Status:          string(app.Status),
			TotalUsage:      usage,
			Prompts:         us.getAppPrompts(ctx, app.ID),
			Uploads:         us.getAppUploads(ctx, app.ID),
			KnowledgeBaseID: app.KnowledgeBaseID,
			Bills:           bills,
		}
		applications = append(applications, newApp)
	}
	// get user total bills
	bills, usage, err := us.getUserAIBills(ctx, oid)
	if err != nil {
		return nil, err
	}
	result := &models.UserAIDetail{
		TotalUsage: usage,
		Apps:       applications,
		Bills:      bills,
	}
	return result, nil
}

func (us *UserService) getApps(ctx context.Context, oid string) ([]*cloud.App, error) {
	query := bson.M{
		"created_by": oid,
	}
	cursor, err := us.appColl.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	apps := make([]*cloud.App, 0)
	for cursor.Next(ctx) {
		app := &cloud.App{}
		if err = cursor.Decode(app); err != nil {
			return nil, err
		}
		apps = append(apps, app)
	}
	return apps, nil
}

func (us *UserService) getAppBills(ctx context.Context, id primitive.ObjectID) ([]models.Bill, uint64, error) {
	bills := make(map[time.Time]models.Bill, 0)
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"app_id", id},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$collected_at"},
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
				{"_id", -1},
			}},
		},
	}
	cursor, err := us.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find ai bills but no documents")
			return []models.Bill{}, 0, nil
		}
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	total := uint64(0)
	for cursor.Next(ctx) {
		var group models.Bill
		if err = cursor.Decode(&group); err != nil {
			return nil, 0, err
		}
		total += group.Usage
		aggregateDailyUsage(bills, group)
	}
	return toFormatBills(bills), total, nil
}

func (us *UserService) getUserAIBills(ctx context.Context, oid string) ([]models.Bill, uint64, error) {
	bills := make(map[time.Time]models.Bill, 0)
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"user_id", oid},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$collected_at"},
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
	cursor, err := us.aiBillColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find ai bills but no documents")
			return []models.Bill{}, 0, nil
		}
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	total := uint64(0)
	for cursor.Next(ctx) {
		var group models.Bill
		if err = cursor.Decode(&group); err != nil {
			return nil, 0, err
		}
		total += group.Usage
		aggregateDailyUsage(bills, group)
	}
	return toFormatBills(bills), total, nil
}

func (us *UserService) getAppUploads(ctx context.Context, id primitive.ObjectID) int64 {
	query := bson.M{
		"app_id": id,
		"status": bson.M{
			"$ne": "deleted",
		},
	}
	cnt, err := us.uploadColl.CountDocuments(ctx, query)
	if err != nil {
		log.Error(ctx).Err(err).Any("id", id).Msg("failed to count app upload")
		return 0
	}
	return cnt
}

func (us *UserService) getAppPrompts(ctx context.Context, id primitive.ObjectID) int64 {
	query := bson.M{
		"app_id": id,
		"status": bson.M{
			"$ne": "deleted",
		},
	}
	cnt, err := us.promptColl.CountDocuments(ctx, query)
	if err != nil {
		log.Error(ctx).Err(err).Any("id", id).Msg("failed to count app prompt")
		return 0
	}
	return cnt
}

func aggregateDailyUsage(bills map[time.Time]models.Bill, bill models.Bill) {
	t := utils.ToBillTimeForAI(bill.Date)
	if val, ok := bills[t]; ok {
		bill.Usage += val.Usage
		bills[t] = bill
	} else {
		bills[t] = bill
	}
}
