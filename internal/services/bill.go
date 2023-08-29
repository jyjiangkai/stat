package services

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"go.mongodb.org/mongo-driver/bson"
)

func (cs *CollectorService) getBills(ctx context.Context, oid string) (*models.Bills, error) {
	aiBill, err := cs.getAIBill(ctx, oid)
	if err != nil {
		return nil, err
	}
	connectBill, err := cs.getConnectBill(ctx, oid)
	if err != nil {
		return nil, err
	}
	return &models.Bills{
		AI:      aiBill,
		Connect: connectBill,
	}, nil
}

func (cs *CollectorService) getConnectBill(ctx context.Context, oid string) (*models.ConnectBills, error) {
	bills, err := cs.getConnectBills(ctx, oid)
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
	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	if value, ok := stat.Items[yesterday]; ok {
		stat.Yesterday = value
	}
	return stat, nil
}

func (cs *CollectorService) getAIBill(ctx context.Context, oid string) (*models.AIBills, error) {
	bills, err := cs.getAIBills(ctx, oid)
	if err != nil {
		return nil, err
	}
	stat := models.NewAIBill()
	for idx := range bills {
		billTime := toBillTimeForAI(bills[idx].CollectedAt)
		if _, ok := stat.Items[billTime]; ok {
			stat.Items[billTime] += bills[idx].Usage.Credits
		} else {
			stat.Items[billTime] = bills[idx].Usage.Credits
		}
		stat.Total += bills[idx].Usage.Credits
	}
	now := time.Now()
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	if value, ok := stat.Items[yesterday]; ok {
		stat.Yesterday = value
	}
	return stat, nil
}

func (cs *CollectorService) getAIBills(ctx context.Context, user string) ([]*cloud.AIBill, error) {
	query := bson.M{
		"user_id": user,
	}
	cursor, err := cs.aiBillColl.Find(ctx, query)
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

func (cs *CollectorService) getConnectBills(ctx context.Context, oid string) ([]*cloud.Bill, error) {
	query := bson.M{
		"user_id": oid,
	}
	cursor, err := cs.billColl.Find(ctx, query)
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
