package bind

import (
	"time"
)

type UserProfile struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	CompanyEmail string    `json:"company_email"`
	GivenName    string    `json:"given_name"`
	FamilyName   string    `json:"family_name"`
	NickName     string    `json:"nickname"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	CompanyName  string    `json:"company_name"`
	OIDC         string    `json:"oidc"`
}

type Bill struct {
	UserID          string            `json:"user_id"`
	ConnectionNum   uint32            `json:"connection_num"`
	ConnectionBills []*ConnectionBill `json:"connection_bills"`
	Usage           *Usage            `json:"usage"`
}

type ConnectionBill struct {
	ConnectionID   string       `json:"connection_id"`
	ConnectionType string       `json:"connection_type"`
	DailyBills     []*DailyBill `json:"daily_bills"`
	Usage          uint64       `json:"usage"`
}

type DailyBill struct {
	Date  time.Time `json:"date"`
	Usage uint64    `json:"usage"`
}

type Usage struct {
	Notice    uint64 `json:"notice"`
	Streaming uint64 `json:"streaming"`
	Total     uint64 `json:"total"`
}
