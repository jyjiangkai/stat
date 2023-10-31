package api

type ListResult struct {
	List []interface{} `json:"list"`
	P    Page          `json:"page"`
}

type Page struct {
	Total      int64  `json:"total" form:"total"`
	PageSize   int64  `json:"page_size" form:"page_size,default=10"`
	PageNumber int64  `json:"page_number" form:"page_number,default=0"`
	SortBy     string `json:"sort_by" form:"sort_by,default=created_at"`
	Direction  string `json:"direction" form:"direction,default=desc"`
	Range      string `json:"range" form:"range"`
	Tag        string `json:"tag" form:"tag"`
}
