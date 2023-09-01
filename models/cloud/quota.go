package cloud

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserQuota struct {
	Base             `json:",inline" bson:",inline"`
	Version          string            `json:"version" bson:"version"`
	Plan             *QuotaPlan        `json:"plan" bson:"plan"`
	QuotaItems       []*Quota          `json:"quota_items" bson:"quota_items"`
	PeriodOfValidity *PeriodOfValidity `json:"period_of_validity" bson:"period_of_validity"`
}

type Price struct {
	CNY float64 `json:"cny" bson:"cny"`
	USD float64 `json:"usd" bson:"usd"`
}

type QuotaPlan struct {
	PlanID primitive.ObjectID `json:"plan_id" bson:"plan_id"`
	Kind   string             `json:"kind" bson:"kind"`
	Type   string             `json:"type" bson:"type"`
	Level  int                `json:"level" bson:"level"`
	Price  *Price             `json:"price" bson:"-"`
}

type ResourceType string

const (
	ResourceConnections     ResourceType = "connections"
	ResourceNoticeEvents    ResourceType = "notice-events"
	ResourceStreamingEvents ResourceType = "streaming-events"
	ResourceApps            ResourceType = "apps"
	ResourceUploads         ResourceType = "uploads"
	ResourceCredits         ResourceType = "credits"
)

type Quota struct {
	Type  ResourceType `json:"type" bson:"type"`
	Used  uint64       `json:"used" bson:"used"`
	Total int64        `json:"total" bson:"total"`
}

type PeriodOfValidity struct {
	Start time.Time `json:"start" bson:"start"`
	End   time.Time `json:"end" bson:"end"`
}
