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
)

func (cs *CollectorService) getClass(ctx context.Context, oid string) (*models.Class, error) {
	aiLevel, err := cs.getLevel(ctx, oid, "ai")
	if err != nil {
		return nil, err
	}
	connectLevel, err := cs.getLevel(ctx, oid, "cloud")
	if err != nil {
		return nil, err
	}
	return &models.Class{
		AI:      aiLevel,
		Connect: connectLevel,
	}, nil
}

func (cs *CollectorService) getLevel(ctx context.Context, oid string, kind string) (*models.Level, error) {
	now := time.Now()
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

	result := cs.quotaColl.FindOne(ctx, query)
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return &models.Level{
				Premium: false,
				Plan: models.Plan{
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
	return &models.Level{
		Premium: true,
		Plan: models.Plan{
			Type:  userQuota.Plan.Type,
			Level: userQuota.Plan.Level,
		},
	}, nil
}
