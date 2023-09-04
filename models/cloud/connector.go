package cloud

import "go.mongodb.org/mongo-driver/bson/primitive"

type Connector struct {
	Base         `json:",inline" bson:",inline"`
	Name         string             `json:"name" bson:"name"`
	ConnectionID primitive.ObjectID `json:"connection_id" bson:"connection_id"`
	Kind         string             `json:"kind" bson:"kind"`
	Type         string             `json:"type" bson:"type"`
	DisplayType  string             `json:"display_type" bson:"display_type"`
	Status       string             `json:"status" bson:"status"`
}
