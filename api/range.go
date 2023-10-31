package api

type Range struct {
	Start string `json:"start" form:"start"`
	End   string `json:"end" form:"end"`
}

type Ranges struct {
	Range Range `json:"range" form:"range"`
}
