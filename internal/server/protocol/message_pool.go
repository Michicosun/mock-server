package protocol

type MessagePool struct {
	PoolName  string `json:"pool_name"`
	QueueName string `json:"queue_name"`
	Broker    string `json:"broker"`
}
