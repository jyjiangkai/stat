package api

type ListResult struct {
	List []interface{} `json:"list"`
	P    Page          `json:"page"`
}

type Page struct {
	Total      int64  `json:"total" form:"total"`
	PageSize   int64  `json:"page_size" form:"page_size,default=20"`
	PageNumber int64  `json:"page_number" form:"page_number,default=1"`
	SortBy     string `json:"sort_by" form:"sort_by,default=updated_at"`
	Direction  string `json:"direction" form:"direction"`
	Reg        string `json:"region,omitempty" form:"region"`
	// Token      string        `json:"token,omitempty"` // reserve
}

// type PageLabel struct {
// 	Page  Page     `json:",inline"`
// 	Label []string `json:"label" form:"label"`
// 	Inner struct {
// 		Label map[string]string
// 	}
// }

// func (p *PageLabel) InitValue() error {
// 	if len(p.Label) > 0 {
// 		label := make(map[string]string, len(p.Label))
// 		for _, l := range p.Label {
// 			str := strings.SplitN(l, ",", 2)
// 			if len(str) != 2 {
// 				return fmt.Errorf("param label value invalid")
// 			}
// 			label[str[0]] = str[1]
// 		}
// 		p.Inner.Label = label
// 	}
// 	return nil
// }

// func (p *PageLabel) GetLabel() map[string]string {
// 	return p.Inner.Label
// }

// func (p Page) GetRegion() cloud.Region {
// 	if p.Reg != "" {
// 		return p.Reg
// 	}
// 	return cloud.DefaultRegion
// }
