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
	Ref          string   `json:"ref" bson:"ref"`
	RefHost      string   `json:"ref_host" bson:"ref_host"`
	Class        *Class   `json:"class" bson:"class"`
	Bills        *Bills   `json:"bills" bson:"bills"`
	Usages       *Usages  `json:"usages" bson:"usages"`
	Cohort       *Cohort  `json:"cohort" bson:"cohort"`
	Credits      *Credits `json:"credits" bson:"credits"`
}

type PremiumUser struct {
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

type Class struct {
	AI      *Level `json:"ai" bson:"ai"`
	Connect *Level `json:"connect" bson:"connect"`
}

type Level struct {
	Premium          bool                    `json:"premium" bson:"premium"`
	Plan             *Plan                   `json:"plan" bson:"plan"`
	Payment          *Payment                `json:"payment" bson:"payment"`
	PeriodOfValidity *cloud.PeriodOfValidity `json:"period_of_validity" bson:"period_of_validity"`
}

type Plan struct {
	Type  string `json:"type" bson:"type"`
	Level int    `json:"level" bson:"level"`
}

type Payment struct {
	cloud.Base `json:",inline" bson:",inline"`
	Desc       string         `json:"desc" bson:"desc"`
	Kind       string         `json:"kind" bson:"kind"`
	Currency   string         `json:"currency" bson:"currency"`
	Amount     *PaymentAmount `json:"amount" bson:"amount"`
}

type PaymentAmount struct {
	Total    float64 `json:"total" bson:"total"`
	Discount float64 `json:"discount" bson:"discount"`
	Payable  float64 `json:"payable" bson:"payable"`
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
	Total     *Events              `json:"total" bson:"total"`
	Yesterday *Events              `json:"yesterday" bson:"yesterday"`
	LastWeek  *Events              `json:"last_week" bson:"last_week"`
	LastMonth *Events              `json:"last_month" bson:"last_month"`
}

type Events struct {
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
	Template      string               `json:"template_id" bson:"template_id"`
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

type UserConversion struct {
	Date       time.Time `json:"date" bson:"date"`
	Tag        string    `json:"tag" bson:"tag"`
	Entered    int64     `json:"entered" bson:"entered"`
	Tried      int64     `json:"tried" bson:"tried"`
	Registered int64     `json:"registered" bson:"registered"`
	Created    int64     `json:"created" bson:"created"`
}

type DailyStatsOfUserNumber struct {
	Date                        time.Time `json:"date" bson:"date"`
	Tag                         string    `json:"tag" bson:"tag"`
	RegisterUserNumber          int64     `json:"register_user_number" bson:"register_user_number"`
	LoginUserNumber             int64     `json:"login_user_number" bson:"login_user_number"`
	ConnectionCreatedUserNumber int64     `json:"connection_created_user_number" bson:"connection_created_user_number"`
	AppCreatedUserNumber        int64     `json:"app_created_user_number" bson:"app_created_user_number"`
	ConnectionUsedUserNumber    int64     `json:"connection_used_user_number" bson:"connection_used_user_number"`
	AppUsedUserNumber           int64     `json:"app_used_user_number" bson:"app_used_user_number"`
}

type DailyStatsOfShopifyLandingPageActionNumber struct {
	Date                                             time.Time `json:"date" bson:"date"`
	Tag                                              string    `json:"tag" bson:"tag"`
	UniqueVisitorNumber                              int64     `json:"unique_visitor_number" bson:"unique_visitor_number"`
	TryVanusActionNumber                             int64     `json:"try_vanus_action_number" bson:"try_vanus_action_number"`
	SignInWithGithubActionNumber                     int64     `json:"sign_in_with_github_action_number" bson:"sign_in_with_github_action_number"`
	SignInWithGoogleActionNumber                     int64     `json:"sign_in_with_google_action_number" bson:"sign_in_with_google_action_number"`
	SignInWithMicrosoftActionNumber                  int64     `json:"sign_in_with_microsoft_action_number" bson:"sign_in_with_microsoft_action_number"`
	ContactUsActionNumber                            int64     `json:"contact_us_action_number" bson:"contact_us_action_number"`
	ShopifyToGoogleSheetsWithNewOrderActionNumber    int64     `json:"shopify_to_googlesheets_with_new_order_action_number" bson:"shopify_to_googlesheets_with_new_order_action_number"`
	ShopifyToGoogleSheetsWithCancelOrderActionNumber int64     `json:"shopify_to_googlesheets_with_cancel_order_action_number" bson:"shopify_to_googlesheets_with_cancel_order_action_number"`
	ShopifyToMailChimpActionNumber                   int64     `json:"shopify_to_mailchimp_action_number" bson:"shopify_to_mailchimp_action_number"`
	ShopifyToMySQLActionNumber                       int64     `json:"shopify_to_mysql_action_number" bson:"shopify_to_mysql_action_number"`
	ShopifyToOutlookWithWelcomeCustomerActionNumber  int64     `json:"shopify_to_outlook_with_welcome_customer_action_number" bson:"shopify_to_outlook_with_welcome_customer_action_number"`
	ShopifyToOutlookWithNewOrderActionNumber         int64     `json:"shopify_to_outlook_with_new_order_action_number" bson:"shopify_to_outlook_with_new_order_action_number"`
	ShopifyToSlackWithNewOrderActionNumber           int64     `json:"shopify_to_slack_with_new_order_action_number" bson:"shopify_to_slack_with_new_order_action_number"`
	ShopifyToSlackWithCancelOrderActionNumber        int64     `json:"shopify_to_slack_with_cancel_order_action_number" bson:"shopify_to_slack_with_cancel_order_action_number"`
}

type DailyStatsOfGithubLandingPageActionNumber struct {
	Date                                        time.Time `json:"date" bson:"date"`
	Tag                                         string    `json:"tag" bson:"tag"`
	UniqueVisitorNumber                         int64     `json:"unique_visitor_number" bson:"unique_visitor_number"`
	TryVanusActionNumber                        int64     `json:"try_vanus_action_number" bson:"try_vanus_action_number"`
	SignInWithGithubActionNumber                int64     `json:"sign_in_with_github_action_number" bson:"sign_in_with_github_action_number"`
	SignInWithGoogleActionNumber                int64     `json:"sign_in_with_google_action_number" bson:"sign_in_with_google_action_number"`
	SignInWithMicrosoftActionNumber             int64     `json:"sign_in_with_microsoft_action_number" bson:"sign_in_with_microsoft_action_number"`
	ContactUsActionNumber                       int64     `json:"contact_us_action_number" bson:"contact_us_action_number"`
	GithubToSlackWithIssueActionNumber          int64     `json:"github_to_slack_with_issue_action_number" bson:"github_to_slack_with_issue_action_number"`
	GithubToSlackWithOpenedPRActionNumber       int64     `json:"github_to_slack_with_opened_pr_action_number" bson:"github_to_slack_with_opened_pr_action_number"`
	GithubToFeishuWithStarActionNumber          int64     `json:"github_to_feishu_with_star_action_number" bson:"github_to_feishu_with_star_action_number"`
	GithubToFeishuWithIssueCommentActionNumber  int64     `json:"github_to_feishu_with_issue_comment_action_number" bson:"github_to_feishu_with_issue_comment_action_number"`
	GithubToGoogleSheetsWithIssueActionNumber   int64     `json:"github_to_google_sheets_with_issue_action_number" bson:"github_to_google_sheets_with_issue_action_number"`
	GithubToDiscordWithIssueCommentActionNumber int64     `json:"github_to_discord_with_issue_comment_action_number" bson:"github_to_discord_with_issue_comment_action_number"`
	GithubToDiscordWithOpenedPRActionNumber     int64     `json:"github_to_discord_with_opened_pr_action_number" bson:"github_to_discord_with_opened_pr_action_number"`
}

type ActiveUserNumber struct {
	Date    time.Time `json:"_id" bson:"_id"`
	Connect int64     `json:"connect" bson:"connect"`
	AI      int64     `json:"ai" bson:"ai"`
}

func NewAIBill() *AIBills {
	return &AIBills{
		Items: make(map[time.Time]uint64),
	}
}

func NewConnectBill() *ConnectBills {
	return &ConnectBills{
		Items:     make(map[time.Time]uint64),
		Total:     &Events{},
		Yesterday: &Events{},
		LastWeek:  &Events{},
		LastMonth: &Events{},
	}
}

func NewPayment() *Payment {
	return &Payment{
		Amount: &PaymentAmount{},
	}
}

// func (daily *DailyStatsOfShopifyLandingPageActionNumber) TotalCount() int64 {
// 	return daily.TryVanusActionNumber + daily.SignInWithGithubActionNumber + daily.SignInWithGoogleActionNumber + daily.SignInWithMicrosoftActionNumber + daily.ContactUsActionNumber + daily.ShopifyToGoogleSheetsWithCancelOrderActionNumber + daily.ShopifyToGoogleSheetsWithNewOrderActionNumber + daily.ShopifyToMailChimpActionNumber + daily.ShopifyToMySQLActionNumber + daily.ShopifyToOutlookWithNewOrderActionNumber + daily.ShopifyToOutlookWithWelcomeCustomerActionNumber + daily.ShopifyToSlackWithCancelOrderActionNumber + daily.ShopifyToSlackWithNewOrderActionNumber
// }

func (daily *DailyStatsOfShopifyLandingPageActionNumber) TriedCount() int64 {
	return daily.TryVanusActionNumber + daily.ShopifyToGoogleSheetsWithCancelOrderActionNumber + daily.ShopifyToGoogleSheetsWithNewOrderActionNumber + daily.ShopifyToMailChimpActionNumber + daily.ShopifyToMySQLActionNumber + daily.ShopifyToOutlookWithNewOrderActionNumber + daily.ShopifyToOutlookWithWelcomeCustomerActionNumber + daily.ShopifyToSlackWithCancelOrderActionNumber + daily.ShopifyToSlackWithNewOrderActionNumber
}

// func (daily *DailyStatsOfGithubLandingPageActionNumber) TotalCount() int64 {
// 	return daily.TryVanusActionNumber + daily.SignInWithGithubActionNumber + daily.SignInWithGoogleActionNumber + daily.SignInWithMicrosoftActionNumber + daily.ContactUsActionNumber + daily.GithubToDiscordWithIssueCommentActionNumber + daily.GithubToDiscordWithOpenedPRActionNumber + daily.GithubToFeishuWithIssueCommentActionNumber + daily.GithubToFeishuWithStarActionNumber + daily.GithubToGoogleSheetsWithIssueActionNumber + daily.GithubToSlackWithIssueActionNumber + daily.GithubToSlackWithOpenedPRActionNumber
// }

func (daily *DailyStatsOfGithubLandingPageActionNumber) TriedCount() int64 {
	return daily.TryVanusActionNumber + daily.GithubToDiscordWithIssueCommentActionNumber + daily.GithubToDiscordWithOpenedPRActionNumber + daily.GithubToFeishuWithIssueCommentActionNumber + daily.GithubToFeishuWithStarActionNumber + daily.GithubToGoogleSheetsWithIssueActionNumber + daily.GithubToSlackWithIssueActionNumber + daily.GithubToSlackWithOpenedPRActionNumber
}
