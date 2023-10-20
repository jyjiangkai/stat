package models

import (
	"time"

	"github.com/jyjiangkai/stat/models/cloud"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Track struct {
	cloud.Base `json:",inline" bson:",inline"`
	OID        string             `json:"oidc_id" bson:"oidc_id"`
	User       string             `json:"user" bson:"user"`
	Type       string             `json:"type" bson:"type"`
	Tag        string             `json:"tag" bson:"tag"`
	Time       time.Time          `json:"time" bson:"time"`
	ActionID   primitive.ObjectID `json:"action_id" bson:"action_id"`
}
