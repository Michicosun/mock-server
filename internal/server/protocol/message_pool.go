package protocol

type MessagePool struct {
	PoolName  string `json:"pool_name" binding:"required,min=1"`
	QueueName string `json:"queue_name,omitempty"`
	TopicName string `json:"topic_name,omitempty"`
	Broker    string `json:"broker" binding:"required,oneof=rabbitmq kafka"`
}
