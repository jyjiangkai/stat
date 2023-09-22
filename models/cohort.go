package models

import (
	"time"

	"github.com/jyjiangkai/stat/models/cloud"
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

type WeeklyRetention struct {
	Week  *Week  `json:"week" bson:"week"`
	Ratio string `json:"ratio" bson:"ratio"`
	Usage uint64 `json:"usage" bson:"usage"`
}

type WeeklyCohortAnalysis struct {
	// WeekStart   string                      `json:"week_start" bson:"week_start"`
	cloud.Base  `json:",inline" bson:",inline"`
	Week        *Week                       `json:"week" bson:"week"`
	TotalUsers  uint64                      `json:"total_users" bson:"total_users"`
	AIRetention map[string]*WeeklyRetention `json:"ai_retention" bson:"ai_retention"`
	CTRetention map[string]*WeeklyRetention `json:"ct_retention" bson:"ct_retention"`
}
