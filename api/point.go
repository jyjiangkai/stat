package api

// {"x":"2023-11-10","y":17,"dataset":"unique visitor","indices":[15]}
type Point struct {
	X       string `json:"x" form:"x"`
	Y       int64  `json:"y" form:"y"`
	Dataset string `json:"dataset" form:"dataset"`
}
