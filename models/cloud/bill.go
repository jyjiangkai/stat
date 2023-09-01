package cloud

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Bill struct {
	ID             primitive.ObjectID `json:"id" bson:"_id"`
	UserID         string             `json:"user_id" bson:"user_id"`
	CollectedAt    time.Time          `json:"collected_at" bson:"collected_at"`
	ConnectionID   primitive.ObjectID `json:"connection_id" bson:"connection_id"`
	ConnectionType string             `json:"connection_type" bson:"connection_type"`
	EventbusID     primitive.ObjectID `json:"eventbus_id" bson:"eventbus_id"`
	ReceivedNum    uint64             `json:"received_num" bson:"received_num"`
	SourceID       primitive.ObjectID `json:"source_id" bson:"source_id"`
	SinkID         primitive.ObjectID `json:"sink_id" bson:"sink_id"`
	UsageNum       uint64             `json:"usage_num" bson:"usage_num"`
}

type AIBill struct {
	ID          primitive.ObjectID `json:"id" bson:"_id"`
	UserID      string             `json:"user_id" bson:"user_id"`
	CollectedAt time.Time          `json:"collected_at" bson:"collected_at"`
	AppID       primitive.ObjectID `json:"app_id" bson:"app_id"`
	AppType     string             `json:"app_type" bson:"app_type"`
	Usage       *Usage             `json:"usage" bson:"usage"`
}

type Usage struct {
	ChatGPT35 uint64 `json:"chatgpt_3_5" bson:"chatgpt_3_5"`
	ChatGPT4  uint64 `json:"chatgpt_4" bson:"chatgpt_4"`
	Credits   uint64 `json:"credits" bson:"-"`
}
