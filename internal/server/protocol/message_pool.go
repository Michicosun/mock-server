package protocol

type MessagePool struct {
	PoolName  string `json:"pool_name"`
	QueueName string `json:"queue_name,omitempty"`
	TopicName string `json:"topic_name,omitempty"`
	Broker    string `json:"broker"`
}
