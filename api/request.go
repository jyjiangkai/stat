package api

type Request struct {
	Point       Point       `json:"point"`
	Range       Range       `json:"range"`
	FilterStack FilterStack `json:"filter_stack"`
}

func NewRequest() Request {
	return Request{
		Point: Point{},
		Range: Range{},
		FilterStack: FilterStack{
			Filters: make([]Filter, 0),
		},
	}
}
