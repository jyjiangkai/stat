package api

type ListOptions struct {
	KindSelector string `json:"kindSelector" form:"kindSelector"`
}

type GetOptions struct {
	KindSelector string `json:"kindSelector" form:"kindSelector"`
}
