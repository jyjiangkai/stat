package api

type Request struct {
	Range       Range       `json:"range"`
	FilterStack FilterStack `json:"filter_stack"`
}

func NewRequest() Request {
	return Request{
		Range: Range{},
		FilterStack: FilterStack{
			Filters: make([]Filter, 0),
		},
	}
}
