package api

type Filter struct {
	Columns  []Column `json:"filters" form:"filters"`
	Operator string   `json:"operator" form:"operator"`
}

type Column struct {
	ColumnID string `json:"columnId" form:"columnId"`
	Operator string `json:"operator" form:"operator"`
	Value    string `json:"value" form:"value"`
}

func NewFilter() Filter {
	return Filter{
		Columns: make([]Column, 0),
	}
}
