package services

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"github.com/jyjiangkai/stat/utils"
	"go.mongodb.org/mongo-driver/bson"
)

func (ss *StatService) getBills(ctx context.Context, oid string, now time.Time) (*models.Bills, error) {
	aiBill, err := ss.getAIBill(ctx, oid, now)
	if err != nil {
		return nil, err
	}
	connectBill, err := ss.getConnectBill(ctx, oid, now)
	if err != nil {
		return nil, err
	}
	return &models.Bills{
		AI:      aiBill,
		Connect: connectBill,
	}, nil
}

func (ss *StatService) getConnectBill(ctx context.Context, oid string, now time.Time) (*models.ConnectBills, error) {
	bills, err := ss.getConnectBills(ctx, oid)
	if err != nil {
		return nil, err
	}
	stat := models.NewConnectBill()
	yest := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	for idx := range bills {
		billTime := utils.ToBillTimeForConnect(bills[idx].CollectedAt)
		if _, ok := stat.Items[billTime]; ok {
			stat.Items[billTime] += bills[idx].UsageNum
		} else {
			stat.Items[billTime] = bills[idx].UsageNum
		}
		if billTime == yest {
			stat.Yesterday.Received += bills[idx].ReceivedNum
			stat.Yesterday.Delivered += bills[idx].DeliveredNum
			stat.Yesterday.Total += bills[idx].UsageNum
		}
		if time.Since(bills[idx].CollectedAt) <= TimeDurationOfWeek {
			stat.LastWeek.Received += bills[idx].ReceivedNum
			stat.LastWeek.Delivered += bills[idx].DeliveredNum
			stat.LastWeek.Total += bills[idx].UsageNum
		}
		if time.Since(bills[idx].CollectedAt) <= TimeDurationOfMonth {
			stat.LastMonth.Received += bills[idx].ReceivedNum
			stat.LastMonth.Delivered += bills[idx].DeliveredNum
			stat.LastMonth.Total += bills[idx].UsageNum
		}
		stat.Total.Received += bills[idx].ReceivedNum
		stat.Total.Delivered += bills[idx].DeliveredNum
		stat.Total.Total += bills[idx].UsageNum
	}
	return stat, nil
}

func (ss *StatService) getAIBill(ctx context.Context, oid string, now time.Time) (*models.AIBills, error) {
	bills, err := ss.getAIBills(ctx, oid)
	if err != nil {
		return nil, err
	}
	stat := models.NewAIBill()
	for idx := range bills {
		credit := bills[idx].Usage.ChatGPT35 + 20*bills[idx].Usage.ChatGPT4
		billTime := utils.ToBillTimeForAI(bills[idx].CollectedAt)
		if _, ok := stat.Items[billTime]; ok {
			stat.Items[billTime] += credit
		} else {
			stat.Items[billTime] = credit
		}
		if time.Since(bills[idx].CollectedAt) <= TimeDurationOfWeek {
			stat.LastWeek += credit
		}
		if time.Since(bills[idx].CollectedAt) <= TimeDurationOfMonth {
			stat.LastMonth += credit
		}
		stat.Total += credit
	}
	yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	if value, ok := stat.Items[yesterday]; ok {
		stat.Yesterday = value
	}
	return stat, nil
}

func (ss *StatService) getAIBills(ctx context.Context, oid string) ([]*cloud.AIBill, error) {
	query := bson.M{
		"user_id": oid,
		// "collected_at": bson.M{
		// 	"$gte": start,
		// 	"$lte": end,
		// },
	}
	cursor, err := ss.aiBillColl.Find(ctx, query)
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

func (ss *StatService) getConnectBills(ctx context.Context, oid string) ([]*cloud.Bill, error) {
	query := bson.M{
		"user_id": oid,
		// "collected_at": bson.M{
		// 	"$gte": start,
		// 	"$lte": end,
		// },
	}
	cursor, err := ss.billColl.Find(ctx, query)
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
