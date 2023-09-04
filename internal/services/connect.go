package services

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/constant"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"github.com/jyjiangkai/stat/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func (us *UserService) getConnectDetail(ctx context.Context, oid string) (*models.UserConnectDetail, error) {
	connections, err := us.getConnections(ctx, oid)
	if err != nil {
		return nil, err
	}
	cns := make([]*models.Connection, 0)
	for idx := range connections {
		c := connections[idx]
		bills, usage, err := us.getConnectionBills(ctx, c.ID)
		if err != nil {
			return nil, err
		}
		cn := &models.Connection{
			Base:          c.Base,
			Name:          c.Name,
			Status:        string(c.Status),
			Description:   c.Description,
			TotalUsage:    usage,
			EventbusID:    c.EventbusID,
			Subscriptions: c.Subscriptions,
			SourceID:      c.SourceID,
			SinkID:        c.SinkID,
			SourceType:    us.getConnectorType(ctx, c.SourceID),
			SinkType:      us.getConnectorType(ctx, c.SinkID),
			Bills:         bills,
		}
		cns = append(cns, cn)
	}
	// get user total bills
	bills, usage, err := us.getUserBills(ctx, oid)
	if err != nil {
		return nil, err
	}
	result := &models.UserConnectDetail{
		TotalUsage:  usage,
		Connections: cns,
		Bills:       bills,
	}
	return result, nil
}

func (us *UserService) getConnections(ctx context.Context, oid string) ([]*cloud.Connection, error) {
	query := bson.M{
		"created_by": oid,
	}
	cursor, err := us.connectionColl.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	connections := make([]*cloud.Connection, 0)
	for cursor.Next(ctx) {
		connection := &cloud.Connection{}
		if err = cursor.Decode(connection); err != nil {
			return nil, err
		}
		connections = append(connections, connection)
	}
	return connections, nil
}

func (us *UserService) getConnectionBills(ctx context.Context, id primitive.ObjectID) ([]models.Bill, uint64, error) {
	bills := make(map[time.Time]models.Bill, 0)
	pipeline := mongo.Pipeline{
		{
			{"$match", bson.D{
				{"connection_id", id},
			}},
		},
		{
			{"$group", bson.D{
				{"_id", "$collected_at"},
				{"usage", bson.D{
					{"$sum", "$usage_num"},
				}},
			}},
		},
	}
	cursor, err := us.billColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find connect bills but no documents")
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
		bills[utils.ToBillTimeForConnect(group.Date)] = group
	}
	return toFormatBills(bills), total, nil
}

func (us *UserService) getUserBills(ctx context.Context, oid string) ([]models.Bill, uint64, error) {
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
					{"$sum", "$usage_num"},
				}},
			}},
		},
	}
	cursor, err := us.billColl.Aggregate(ctx, pipeline)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Warn(ctx).Err(err).Msg("find connect bills but no documents")
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
		bills[utils.ToBillTimeForConnect(group.Date)] = group
	}
	return toFormatBills(bills), total, nil
}

func (us *UserService) getConnectorType(ctx context.Context, id primitive.ObjectID) string {
	query := bson.M{
		"_id": id,
	}

	result := us.connectorColl.FindOne(ctx, query)
	if result.Err() != nil {
		log.Error(ctx).Err(result.Err()).Any("id", id).Msg("failed to get connector")
		return ""
	}
	connector := &cloud.Connector{}
	if err := result.Decode(connector); err != nil {
		log.Error(ctx).Err(err).Msg("failed to decode user quota")
		return ""
	}
	if connector.DisplayType != "" {
		return connector.DisplayType
	}
	return connector.Type
}

func toFormatBills(billMap map[time.Time]models.Bill) []models.Bill {
	bills := make([]models.Bill, 0)
	rt := time.Now().Add(-24 * time.Hour)
	collectionTime := time.Date(rt.Year(), rt.Month(), rt.Day(), 0, 0, 0, 0, time.UTC)
	for i := 0; i < constant.NumberOfHistogramSamples; i++ {
		if bill, ok := billMap[collectionTime]; ok {
			bills = append(bills, bill)
		} else {
			bills = append(bills, models.Bill{
				Date:  collectionTime,
				Usage: 0,
			})
		}
		collectionTime = collectionTime.Add(-24 * time.Hour)
	}
	return bills
}
