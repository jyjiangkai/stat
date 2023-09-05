package models

import (
	"time"

	"github.com/jyjiangkai/stat/models/cloud"
)

type CohortAnalysis struct {
	cloud.Base `json:",inline" bson:",inline"`
	UserNum    int64        `json:"user_num" bson:"user_num"`
	Week       *Week        `json:"week" bson:"week"`
	AI         []*Retention `json:"ai" bson:"ai"`
	Connect    []*Retention `json:"connect" bson:"connect"`
}

type Week struct {
	Number int64     `json:"number" bson:"number"`
	Alias  string    `json:"alias" bson:"alias"`
	Start  time.Time `json:"start" bson:"start"`
	End    time.Time `json:"end" bson:"end"`
}

type Retention struct {
	Week  *Week  `json:"week" bson:"week"`
	Rate  string `json:"rate" bson:"rate"`
	Usage uint64 `json:"usage" bson:"usage"`
}
