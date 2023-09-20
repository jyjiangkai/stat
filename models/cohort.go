package models

import (
	"time"
)

// 用于记录一个用户的所有cohort数据
type Cohort struct {
	// 指该用户的创建时间是数据哪一周的
	Week *Week `json:"week" bson:"week"`
	// 指该用户的AI留存数据
	AI map[string]*Retention `json:"ai" bson:"ai"`
	// 指该用户的Connect留存数据
	Connect map[string]*Retention `json:"connect" bson:"connect"`
}

type Week struct {
	Number int64     `json:"number" bson:"number"`
	Alias  string    `json:"alias" bson:"alias"`
	Start  time.Time `json:"start" bson:"start"`
	End    time.Time `json:"end" bson:"end"`
}

type Retention struct {
	Week   *Week  `json:"week" bson:"week"`
	Active bool   `json:"active" bson:"active"`
	Usage  uint64 `json:"usage" bson:"usage"`
}

// Items     map[time.Time]uint64 `json:"items" bson:"items"`
