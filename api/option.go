package api

type ListOptions struct {
	KindSelector string `json:"kindSelector" form:"kindSelector"`
	TypeSelector string `json:"typeSelector" form:"typeSelector"`
}

type GetOptions struct {
	KindSelector string `json:"kindSelector" form:"kindSelector"`
}
