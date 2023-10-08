package cloud

type App struct {
	Base            `json:",inline" bson:",inline"`
	Name            string   `json:"name" bson:"name"`
	Type            string   `json:"type" bson:"type"`
	Model           string   `json:"model" bson:"model"`
	Status          Status   `json:"status" bson:"status"`
	Greeting        string   `json:"greeting,omitempty" bson:"greeting"`
	Prompt          string   `json:"prompt,omitempty" bson:"prompt"`
	KnowledgeBaseID []string `json:"-" bson:"knowledge_base_id"`
}
