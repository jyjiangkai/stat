package services

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/db"
	"github.com/jyjiangkai/stat/log"
	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (rs *RefreshService) getClass(ctx context.Context, oid string, now time.Time) (*models.Class, error) {
	aiLevel, err := rs.getLevel(ctx, oid, "ai", now)
	if err != nil {
		return nil, err
	}
	connectLevel, err := rs.getLevel(ctx, oid, "cloud", now)
	if err != nil {
		return nil, err
	}
	return &models.Class{
		AI:      aiLevel,
		Connect: connectLevel,
	}, nil
}

func (rs *RefreshService) getLevel(ctx context.Context, oid string, kind string, now time.Time) (*models.Level, error) {
	query := bson.M{
		"created_by": oid,
		"plan.kind":  kind,
		"period_of_validity.start": bson.M{
			"$lte": now,
		},
		"period_of_validity.end": bson.M{
			"$gte": now,
		},
	}
	result := rs.quotaColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return &models.Level{
				Premium: false,
				Plan: &models.Plan{
					Type:  "Free",
					Level: 1,
				},
			}, nil
		}
	}
	userQuota := &cloud.UserQuota{}
	if err := result.Decode(userQuota); err != nil {
		log.Error(ctx).Err(err).Msg("failed to decode user quota")
		return nil, db.HandleDBError(err)
	}

	payment, err := rs.getPayment(ctx, oid, kind)
	if err != nil {
		log.Error(ctx).Err(err).Msg("failed to get payment")
		return nil, db.HandleDBError(err)
	}
	level := &models.Level{
		Premium: true,
		Plan: &models.Plan{
			Type:  userQuota.Plan.Type,
			Level: userQuota.Plan.Level,
		},
		Payment:          payment,
		PeriodOfValidity: userQuota.PeriodOfValidity,
	}
	if payment.Currency == "" {
		level.Premium = false
	}
	return level, nil
}

func (rs *RefreshService) getPayment(ctx context.Context, oid string, kind string) (*models.Payment, error) {
	query := bson.M{
		"created_by": oid,
		"kind":       kind,
	}
	opt := options.FindOneOptions{
		Sort: bson.M{"created_at": -1},
	}
	result := rs.paymentColl.FindOne(ctx, query, &opt)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return models.NewPayment(), nil
		}
	}
	payment := &models.Payment{}
	if err := result.Decode(payment); err != nil {
		log.Error(ctx).Err(err).Msg("failed to decode payment")
		return nil, db.HandleDBError(err)
	}
	return payment, nil
}
