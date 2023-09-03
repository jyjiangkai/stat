package cloud

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Status string

func NewBase(ctx context.Context) Base {
	return Base{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: utils.GetUserID(ctx),
		UpdatedBy: utils.GetUserID(ctx),
	}
}

type Base struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at,omitempty" bson:"updated_at"`
	CreatedBy string             `json:"created_by,omitempty" bson:"created_by"`
	UpdatedBy string             `json:"updated_by,omitempty" bson:"updated_by"`
	Reg       string             `json:"region,omitempty" bson:"region"`
}
