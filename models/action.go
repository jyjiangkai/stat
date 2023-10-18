package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Action struct {
	ID       primitive.ObjectID `json:"id" bson:"_id"`
	OID      string             `json:"usersub" bson:"usersub"`
	User     string             `json:"user" bson:"user"`
	Type     string             `json:"type" bson:"type"`
	Action   string             `json:"action" bson:"action"`
	Url      string             `json:"url" bson:"url"`
	Source   string             `json:"source" bson:"source"`
	Website  string             `json:"website" bson:"website"`
	Template string             `json:"template" bson:"template"`
	Time     time.Time          `json:"time" bson:"time"`
	Payload  *Payload           `json:"payload" bson:"payload"`
	App      *ActionApp         `json:"app" bson:"app"`
}

type Payload struct {
	AppID   string `json:"applicationId" bson:"applicationId"`
	Message string `json:"message" bson:"message"`
	Lang    string `json:"lang" bson:"lang"`
	Model   string `json:"model" bson:"model"`
	From    string `json:"from" bson:"from"`
}

type ActionApp struct {
	Name     string `json:"name" bson:"name"`
	Type     string `json:"type" bson:"type"`
	Model    string `json:"model" bson:"model"`
	Status   string `json:"status" bson:"status"`
	Greeting string `json:"greeting,omitempty" bson:"greeting"`
	Prompt   string `json:"prompt,omitempty" bson:"prompt"`
}

func NewActionApp() *ActionApp {
	return &ActionApp{}
}
