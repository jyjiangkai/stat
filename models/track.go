package models

import (
	"time"
)

type Track struct {
	User  string    `json:"user" bson:"user"`
	Tag   string    `json:"tag" bson:"tag"`
	Count uint64    `json:"count" bson:"count"`
	Time  time.Time `json:"time" bson:"time"`
}
