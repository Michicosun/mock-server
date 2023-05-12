package protocol

type EsbRecord struct {
	PoolNameIn  string `json:"pool_name_in" binding:"required"`
	PoolNameOut string `json:"pool_name_out" binding:"required"`
	Code        string `json:"code,omitempty"`
}
