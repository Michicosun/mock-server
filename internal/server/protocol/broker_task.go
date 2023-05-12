package protocol

type BrokerTask struct {
	PoolName string   `json:"pool_name" binding:"required"`
	Messages []string `json:"messages" binding:"required"`
}
