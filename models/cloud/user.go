package cloud

type User struct {
	Base          `json:",inline" bson:",inline"`
	OID           string                 `json:"oidc_id" bson:"oidc_id"`
	Profile       map[string]interface{} `json:"profile" bson:"profile"`
	Email         string                 `json:"email" bson:"email"`
	CompanyEmail  string                 `json:"company_email" bson:"company_email"`
	Phone         string                 `json:"phone" bson:"phone"`
	Country       string                 `json:"country" bson:"country"`
	Verified      bool                   `json:"verified" bson:"verified"`
	GivenName     string                 `json:"given_name" bson:"given_name"`
	FamilyName    string                 `json:"family_name" bson:"family_name"`
	NickName      string                 `json:"nickname" bson:"nickname"`
	CompanyName   string                 `json:"company_name" bson:"company_name"`
	Industry      string                 `json:"industry" bson:"industry"`
	IndustryExtra string                 `json:"industry_extra" bson:"industry_extra"`
}
