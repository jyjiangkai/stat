package services

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"github.com/jyjiangkai/stat/utils"
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
	var (
		total     = &models.Events{}
		yesterday = &models.Events{}
		week      = &models.Events{}
		month     = &models.Events{}
	)
	bills, err := rs.getConnectBills(ctx, oid)
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
			yesterday.Received += bills[idx].ReceivedNum
			yesterday.Delivered += bills[idx].DeliveredNum
			yesterday.Total += bills[idx].UsageNum
		}
		if time.Since(bills[idx].CollectedAt) <= TimeDurationOfWeek {
			week.Received += bills[idx].ReceivedNum
			week.Total += bills[idx].UsageNum
		}
		if time.Since(bills[idx].CollectedAt) <= TimeDurationOfMonth {
			month.Received += bills[idx].ReceivedNum
			month.Total += bills[idx].UsageNum
		}
		total.Received += bills[idx].ReceivedNum
		total.Total += bills[idx].UsageNum
	}
	// yesterday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).Add(-24 * time.Hour)
	// if value, ok := stat.Items[yesterday]; ok {
	// 	stat.Yesterday = value
	// }
	total.Delivered = total.Total - total.Received
	week.Delivered = week.Total - week.Received
	month.Delivered = month.Total - month.Received
	stat.Total = total
	stat.Yesterday = yesterday
	stat.LastWeek = week
	stat.LastMonth = month
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
