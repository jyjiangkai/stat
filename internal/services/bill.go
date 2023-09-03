package services

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"go.mongodb.org/mongo-driver/bson"
)

func (rs *RefreshService) getBills(ctx context.Context, oid string, now time.Time) (*models.Bills, error) {
	aiBill, err := rs.getAIBill(ctx, oid, now)
	if err != nil {
		return nil, err
	}
	connectBill, err := rs.getConnectBill(ctx, oid, now)
	if err != nil {
		return nil, err
	}
	return &models.Bills{
		AI:      aiBill,
		Connect: connectBill,
	}, nil
}

func (rs *RefreshService) getConnectBill(ctx context.Context, oid string, now time.Time) (*models.ConnectBills, error) {
	bills, err := rs.getConnectBills(ctx, oid)
	if err != nil {
		return nil, err
	}
	stat := models.NewConnectBill()
	for idx := range bills {
		billTime := toBillTimeForConnect(bills[idx].CollectedAt)
		if _, ok := stat.Items[billTime]; ok {
			stat.Items[billTime] += bills[idx].UsageNum
		} else {
			stat.Items[billTime] = bills[idx].UsageNum
		}
		stat.Total += bills[idx].UsageNum
	}
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	if value, ok := stat.Items[yesterday]; ok {
		stat.Yesterday = value
	}
	return stat, nil
}

func (rs *RefreshService) getAIBill(ctx context.Context, oid string, now time.Time) (*models.AIBills, error) {
	bills, err := rs.getAIBills(ctx, oid)
	if err != nil {
		return nil, err
	}
	stat := models.NewAIBill()
	for idx := range bills {
		credit := bills[idx].Usage.ChatGPT35 + 20*bills[idx].Usage.ChatGPT4
		billTime := toBillTimeForAI(bills[idx].CollectedAt)
		if _, ok := stat.Items[billTime]; ok {
			stat.Items[billTime] += credit
		} else {
			stat.Items[billTime] = credit
		}
		stat.Total += credit
	}
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	if value, ok := stat.Items[yesterday]; ok {
		stat.Yesterday = value
	}
	return stat, nil
}

func (rs *RefreshService) getAIBills(ctx context.Context, oid string) ([]*cloud.AIBill, error) {
	query := bson.M{
		"user_id": oid,
		// "collected_at": bson.M{
		// 	"$gte": start,
		// 	"$lte": end,
		// },
	}
	cursor, err := rs.aiBillColl.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	bills := make([]*cloud.AIBill, 0)
	for cursor.Next(ctx) {
		bill := &cloud.AIBill{}
		if err = cursor.Decode(bill); err != nil {
			return nil, err
		}
		bills = append(bills, bill)
	}
	return bills, nil
}

func (rs *RefreshService) getConnectBills(ctx context.Context, oid string) ([]*cloud.Bill, error) {
	query := bson.M{
		"user_id": oid,
		// "collected_at": bson.M{
		// 	"$gte": start,
		// 	"$lte": end,
		// },
	}
	cursor, err := rs.billColl.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	bills := make([]*cloud.Bill, 0)
	for cursor.Next(ctx) {
		bill := &cloud.Bill{}
		if err = cursor.Decode(bill); err != nil {
			return nil, err
		}
		bills = append(bills, bill)
	}
	return bills, nil
}

func toBillTimeForConnect(t time.Time) time.Time {
	return t.Add(-24 * time.Hour)
}

func toBillTimeForAI(t time.Time) time.Time {
	var real time.Time
	if t.Hour() == 0 {
		real = t.Add(-24 * time.Hour)
	} else {
		real = t
	}
	return time.Date(real.Year(), real.Month(), real.Day(), 0, 0, 0, 0, time.UTC)
}
