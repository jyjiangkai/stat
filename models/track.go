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

type WeeklyUserTrack struct {
	Week                 *Week  `json:"week" bson:"week"`
	Tag                  string `json:"tag" bson:"tag"`
	UserNum              int64  `json:"user_num" bson:"user_num"`
	LoginNum             int64  `json:"login_num" bson:"login_num"`
	HighKnowledgeBaseNum int64  `json:"high_knowledge_base_num" bson:"high_knowledge_base_num"`
	ViewPriceNum         int64  `json:"view_price_num" bson:"view_price_num"`
	PremiumNum           int64  `json:"premium_num" bson:"premium_num"`
}
