package protocol

type EsbRecord struct {
	PoolNameIn  string `json:"pool_name_in"`
	PoolNameOut string `json:"pool_name_out"`
	Code        string `json:"code"`
}
