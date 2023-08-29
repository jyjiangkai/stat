package cloud

type App struct {
	Base            `json:",inline" bson:",inline"`
	Name            string   `json:"name" bson:"name"`
	Type            string   `json:"type" bson:"type"`
	Model           string   `json:"model" bson:"model"`
	Status          Status   `json:"status" bson:"status"`
	KnowledgeBaseID []string `json:"-" bson:"knowledge_base_id"`
}
