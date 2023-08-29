package cloud

import (
	"context"
	"time"

	"github.com/jyjiangkai/stat/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

const (
	UserKindOfActive  string = "active"
	UserKindOfCreated string = "created"
	UserKindOfUsed    string = "used"
	UserKindOfPaid    string = "paid"
)

type Status string

func NewBase(ctx context.Context) Base {
	return Base{
		ID:        primitive.NewObjectID(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		CreatedBy: utils.GetUserID(ctx),
		UpdatedBy: utils.GetUserID(ctx),
	}
}

type Base struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time          `json:"updated_at,omitempty" bson:"updated_at"`
	CreatedBy string             `json:"created_by,omitempty" bson:"created_by"`
	UpdatedBy string             `json:"updated_by,omitempty" bson:"updated_by"`
	Reg       string             `json:"region,omitempty" bson:"region"`
}

// func (b Base) GetRegion() Region {
// 	if b.Reg != "" {
// 		return b.Reg
// 	}
// 	return DefaultRegion
// }

func (b Base) ObjectID() primitive.ObjectID {
	return b.ID
}

type Organization struct {
	Base `json:",inline" bson:",inline"`
	Name string `json:"name" bson:"name"`
}

type Statistic struct {
	Base         `json:",inline" bson:",inline"`
	OID          string     `json:"oidc_id" bson:"oidc_id"`
	Phone        string     `json:"phone" bson:"phone"`
	Email        string     `json:"email" bson:"email"`
	Country      string     `json:"country" bson:"country"`
	GivenName    string     `json:"given_name" bson:"given_name"`
	FamilyName   string     `json:"family_name" bson:"family_name"`
	NickName     string     `json:"nickname" bson:"nickname"`
	CompanyName  string     `json:"company_name" bson:"company_name"`
	CompanyEmail string     `json:"company_email" bson:"company_email"`
	Class        *Class     `json:"class" bson:"class"`
	Bill         *BillStat  `json:"bill_stat" bson:"bill_stat"`
	Usage        *UsageStat `json:"usage_stat" bson:"usage_stat"`
}

type Class struct {
	AI      *Level `json:"ai_level" bson:"ai_level"`
	Connect *Level `json:"connect_level" bson:"connect_level"`
}

type Level struct {
	Premium bool `json:"premium" bson:"premium"`
	Plan    Plan `json:"plan" bson:"plan"`
}

type Plan struct {
	Type  string `json:"type" bson:"type"`
	Level int    `json:"level" bson:"level"`
}

type BillStat struct {
	AI      *AIBillStat      `json:"ai_bill_stat" bson:"ai_bill_stat"`
	Connect *ConnectBillStat `json:"connect_bill_stat" bson:"connect_bill_stat"`
}

type AIBillStat struct {
	Items     map[time.Time]uint64 `json:"items" bson:"items"`
	Total     uint64               `json:"total" bson:"total"`
	Yesterday uint64               `json:"yesterday" bson:"yesterday"`
}

type ConnectBillStat struct {
	Items     map[time.Time]uint64 `json:"items" bson:"items"`
	Total     uint64               `json:"total" bson:"total"`
	Yesterday uint64               `json:"yesterday" bson:"yesterday"`
}

type UsageStat struct {
	AI      AIUsageStat      `json:"ai_usage" bson:"ai_usage"`
	Connect ConnectUsageStat `json:"connect_usage" bson:"connect_usage"`
}

type AIUsageStat struct {
	App           int64 `json:"app" bson:"app"`
	Upload        int64 `json:"upload" bson:"upload"`
	KnowledgeBase int64 `json:"knowledge_base" bson:"knowledge_base"`
}

type ConnectUsageStat struct {
	Connection int64 `json:"connection" bson:"connection"`
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

func NewAIBillStat() *AIBillStat {
	return &AIBillStat{
		Items: make(map[time.Time]uint64),
	}
}

func NewConnectBillStat() *ConnectBillStat {
	return &ConnectBillStat{
		Items: make(map[time.Time]uint64),
	}
}
