package server

import "github.com/gin-gonic/gin"

func (s *server) initBrokersApiEsb(brokersApi *gin.RouterGroup) {
	esbBrokersEndpoint := "/esb"

	// get all tasks by esb pair in-pool name
	brokersApi.GET(esbBrokersEndpoint, func(c *gin.Context) {})

	// create new esb pair
	brokersApi.POST(esbBrokersEndpoint, func(c *gin.Context) {})

	// create delete esb pair
	brokersApi.DELETE(esbBrokersEndpoint, func(c *gin.Context) {})
}
