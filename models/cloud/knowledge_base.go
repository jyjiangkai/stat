package cloud

type KnowledgeBase struct {
	Base `json:",inline" bson:",inline"`
	Name string `json:"name" bson:"name"`
}
