package protocol

type BrokerTask struct {
	PoolName string   `json:"pool_name"`
	Messages []string `json:"messages"`
}
