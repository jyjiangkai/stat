package models

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/models/cloud"
	"github.com/jyjiangkai/stat/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func NewBase(ctx context.Context) cloud.Base {
	return cloud.Base{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: utils.GetUserID(ctx),
		UpdatedBy: utils.GetUserID(ctx),
	}
}

type Organization struct {
	cloud.Base `json:",inline" bson:",inline"`
	Name       string `json:"name" bson:"name"`
}

type User struct {
	cloud.Base   `json:",inline" bson:",inline"`
	OID          string   `json:"oidc_id" bson:"oidc_id"`
	Phone        string   `json:"phone" bson:"phone"`
	Email        string   `json:"email" bson:"email"`
	Country      string   `json:"country" bson:"country"`
	GivenName    string   `json:"given_name" bson:"given_name"`
	FamilyName   string   `json:"family_name" bson:"family_name"`
	NickName     string   `json:"nickname" bson:"nickname"`
	CompanyName  string   `json:"company_name" bson:"company_name"`
	CompanyEmail string   `json:"company_email" bson:"company_email"`
	Industry     string   `json:"industry" bson:"industry"`
	Class        *Class   `json:"class" bson:"class"`
	Bills        *Bills   `json:"bills" bson:"bills"`
	Usages       *Usages  `json:"usages" bson:"usages"`
	Cohort       *Cohort  `json:"cohort" bson:"cohort"`
	Credits      *Credits `json:"credits" bson:"credits"`
}

// type PremiumUser struct {
// 	cloud.Base   `json:",inline" bson:",inline"`
// 	OID          string   `json:"oidc_id" bson:"oidc_id"`
// 	Phone        string   `json:"phone" bson:"phone"`
// 	Email        string   `json:"email" bson:"email"`
// 	Country      string   `json:"country" bson:"country"`
// 	GivenName    string   `json:"given_name" bson:"given_name"`
// 	FamilyName   string   `json:"family_name" bson:"family_name"`
// 	NickName     string   `json:"nickname" bson:"nickname"`
// 	CompanyName  string   `json:"company_name" bson:"company_name"`
// 	CompanyEmail string   `json:"company_email" bson:"company_email"`
// 	Class        *Class   `json:"class" bson:"class"`
// 	Bills        *Bills   `json:"bills" bson:"bills"`
// 	Usages       *Usages  `json:"usages" bson:"usages"`
// 	Cohort       *Cohort  `json:"cohort" bson:"cohort"`
// 	Credits      *Credits `json:"credits" bson:"credits"`
// }

type Class struct {
	AI      *Level `json:"ai" bson:"ai"`
	Connect *Level `json:"connect" bson:"connect"`
}

type Level struct {
	Premium bool `json:"premium" bson:"premium"`
	Plan    Plan `json:"plan" bson:"plan"`
}

type Plan struct {
	Type  string `json:"type" bson:"type"`
	Level int    `json:"level" bson:"level"`
}

type Bills struct {
	AI      *AIBills      `json:"ai" bson:"ai"`
	Connect *ConnectBills `json:"connect" bson:"connect"`
}

type AIBills struct {
	Items     map[time.Time]uint64 `json:"items" bson:"items"`
	Total     uint64               `json:"total" bson:"total"`
	Yesterday uint64               `json:"yesterday" bson:"yesterday"`
	LastWeek  uint64               `json:"last_week" bson:"last_week"`
	LastMonth uint64               `json:"last_month" bson:"last_month"`
}

type ConnectBills struct {
	Items     map[time.Time]uint64 `json:"items" bson:"items"`
	Total     uint64               `json:"total" bson:"total"`
	Yesterday uint64               `json:"yesterday" bson:"yesterday"`
	LastWeek  uint64               `json:"last_week" bson:"last_week"`
	LastMonth uint64               `json:"last_month" bson:"last_month"`
}

type Usage struct {
	Received  uint64 `json:"received" bson:"received"`
	Delivered uint64 `json:"delivered" bson:"delivered"`
	Total     uint64 `json:"total" bson:"total"`
}

type Usages struct {
	AI      *AIUsages      `json:"ai" bson:"ai"`
	Connect *ConnectUsages `json:"connect" bson:"connect"`
}

type AIUsages struct {
	App           int64 `json:"app" bson:"app"`
	Upload        int64 `json:"upload" bson:"upload"`
	KnowledgeBase int64 `json:"knowledge_base" bson:"knowledge_base"`
}

type ConnectUsages struct {
	Connection int64 `json:"connection" bson:"connection"`
}

type Credits struct {
	Used     uint64 `json:"used" bson:"used"`
	Total    uint64 `json:"total" bson:"total"`
	UsageStr string `json:"usage_str" bson:"usage_str"`
}

type UserDetail struct {
	AI      *UserAIDetail      `json:"ai" bson:"ai"`
	Connect *UserConnectDetail `json:"connect" bson:"connect"`
}

type UserAIDetail struct {
	TotalUsage uint64 `json:"total_usage" bson:"total_usage"`
	Apps       []*App `json:"apps" bson:"apps"`
	Bills      []Bill `json:"bills" bson:"bills"`
}

type UserConnectDetail struct {
	TotalUsage  uint64        `json:"total_usage" bson:"total_usage"`
	Connections []*Connection `json:"connections" bson:"connections"`
	Bills       []Bill        `json:"bills" bson:"bills"`
}

type App struct {
	cloud.Base      `json:",inline" bson:",inline"`
	Name            string   `json:"name" bson:"name"`
	Type            string   `json:"type" bson:"type"`
	Model           string   `json:"model" bson:"model"`
	Status          string   `json:"status" bson:"status"`
	TotalUsage      uint64   `json:"total_usage" bson:"total_usage"`
	Prompts         int64    `json:"prompts" bson:"prompts"`
	Uploads         int64    `json:"uploads" bson:"uploads"`
	KnowledgeBaseID []string `json:"knowledge_base_id" bson:"knowledge_base_id"`
	Bills           []Bill   `json:"bills" bson:"bills"`
}

type Connection struct {
	cloud.Base    `json:",inline" bson:",inline"`
	Name          string               `json:"name" bson:"name"`
	Status        string               `json:"status" bson:"status"`
	Description   string               `json:"description" bson:"description"`
	TotalUsage    uint64               `json:"total_usage" bson:"total_usage"`
	EventbusID    primitive.ObjectID   `json:"eventbus_id" bson:"eventbus_id"`
	Subscriptions []primitive.ObjectID `json:"subscriptions" bson:"subscriptions"`
	SourceID      primitive.ObjectID   `json:"source_id" bson:"source_id"`
	SinkID        primitive.ObjectID   `json:"sink_id" bson:"sink_id"`
	SourceType    string               `json:"source_type" bson:"source_type"`
	SinkType      string               `json:"sink_type" bson:"sink_type"`
	Bills         []Bill               `json:"bills" bson:"bills"`
}

type Bill struct {
	Date  time.Time `json:"_id" bson:"_id"`
	Usage uint64    `json:"usage" bson:"usage"`
}

type RegionInfo struct {
	Name                   string `bson:"name"`
	Provider               string `bson:"provider"`
	Location               string `bson:"location"`
	GatewayEndpoint        string `bson:"gateway_endpoint"`
	OperatorEndpoint       string `bson:"operator_endpoint"`
	PrometheusEndpoint     string `bson:"prometheus_endpoint"`
	Token                  string `bson:"token"`
	ExternalDNS            string `bson:"external_dns"`
	IntegrationExternalDNS string `bson:"integration_external_dns"`
}

func NewAIBill() *AIBills {
	return &AIBills{
		Items: make(map[time.Time]uint64),
	}
}

func NewConnectBill() *ConnectBills {
	return &ConnectBills{
		Items: make(map[time.Time]uint64),
	}
}
