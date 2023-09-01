package cloud

import "go.mongodb.org/mongo-driver/bson/primitive"

type Connection struct {
	Base          `json:",inline" bson:",inline"`
	Name          string               `json:"name" bson:"name"`
	Status        Status               `json:"status" bson:"status"`
	Description   string               `json:"description" bson:"description"`
	EventbusID    primitive.ObjectID   `json:"eventbus_id" bson:"eventbus_id"`
	Subscriptions []primitive.ObjectID `json:"subscriptions" bson:"subscriptions"`
	SourceID      primitive.ObjectID   `json:"source_id" bson:"source_id"`
	SinkID        primitive.ObjectID   `json:"sink_id" bson:"sink_id"`
}
