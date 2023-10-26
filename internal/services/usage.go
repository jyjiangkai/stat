package services

import (
	"context"

	"github.com/jyjiangkai/stat/models"
	"github.com/jyjiangkai/stat/models/cloud"
	"go.mongodb.org/mongo-driver/bson"
)

func (ss *StatService) getUsages(ctx context.Context, oid string) (*models.Usages, error) {
	aiUsages, err := ss.getAIUsages(ctx, oid)
	if err != nil {
		return nil, err
	}
	connectUsages, err := ss.getConnectUsages(ctx, oid)
	if err != nil {
		return nil, err
	}
	return &models.Usages{
		AI:      aiUsages,
		Connect: connectUsages,
	}, nil
}

func (ss *StatService) getAIUsages(ctx context.Context, oid string) (*models.AIUsages, error) {
	usages := &models.AIUsages{}
	uploadQuery := bson.M{
		"created_by": oid,
		"status": bson.M{
			"$ne": "deleted",
		},
	}
	uploadNum, err := ss.aiUploadColl.CountDocuments(ctx, uploadQuery)
	if err != nil {
		return nil, err
	}
	usages.Upload = uploadNum

	appQuery := bson.M{
		"created_by": oid,
		"status": bson.M{
			"$ne": "deleted",
		},
	}
	cursor, err := ss.aiAppColl.Find(ctx, appQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = cursor.Close(ctx)
	}()

	appNum := int64(0)
	knowledgeBaseNum := 0
	for cursor.Next(ctx) {
		app := &cloud.App{}
		if err = cursor.Decode(app); err != nil {
			return nil, err
		}
		appNum += 1
		if app.KnowledgeBaseID != nil {
			knowledgeBaseNum += len(app.KnowledgeBaseID)
		}
	}
	usages.App = appNum
	usages.KnowledgeBase = int64(knowledgeBaseNum)
	return usages, nil
}

func (ss *StatService) getConnectUsages(ctx context.Context, oid string) (*models.ConnectUsages, error) {
	usages := &models.ConnectUsages{}
	query := bson.M{
		"created_by": oid,
		"status": bson.M{
			"$ne": "deleted",
		},
	}
	num, err := ss.connectionColl.CountDocuments(ctx, query)
	if err != nil {
		return nil, err
	}
	usages.Connection = num
	return usages, nil
}
