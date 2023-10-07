package cloud

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserCredits struct {
	ID               primitive.ObjectID `bson:"_id"`
	CreatedAt        time.Time          `bson:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at"`
	QuotaID          primitive.ObjectID `json:"-" bson:"quota_id"`
	UserID           string             `bson:"user_id"`
	Kind             string             `bson:"kind"`
	Type             CreditsType        `json:"type" bson:"type"`
	Used             uint64             `json:"used" bson:"used"`
	Total            int64              `json:"total" bson:"total"`
	Status           string             `json:"status" bson:"status"`
	PeriodOfValidity *PeriodOfValidity  `json:"period_of_validity" bson:"period_of_validity"`
}

type CreditsType string

const (
	FreeUserCredits CreditsType = "free"
	NewUserCredits  CreditsType = "newUser"
	MonthlyCredits  CreditsType = "monthly"
)

type UserCreditsList []*UserCredits

func (list UserCreditsList) ExistMonthlyCredits() bool {
	if len(list) == 0 {
		return false
	}
	for _, c := range list {
		if c.Type == MonthlyCredits {
			return true
		}
	}
	return false
}

func (list UserCreditsList) ExistFreeCredits() bool {
	if len(list) == 0 {
		return false
	}
	for _, c := range list {
		if c.Type == FreeUserCredits {
			return true
		}
	}
	return false
}

func (list UserCreditsList) IndexFreeCredits() int {
	for i, c := range list {
		if c.Type == FreeUserCredits {
			return i
		}
	}
	return -1
}

func (list UserCreditsList) ExistNewUserCredits() bool {
	if len(list) == 0 {
		return false
	}
	for _, c := range list {
		if c.Type == NewUserCredits {
			return true
		}
	}
	return false
}

func (list UserCreditsList) GetPlanPeriod() *PeriodOfValidity {
	if len(list) == 0 {
		return nil
	}
	for _, c := range list {
		if c.Type == MonthlyCredits {
			return c.PeriodOfValidity
		}
		if c.Type == FreeUserCredits {
			return c.PeriodOfValidity
		}
	}
	return nil
}
