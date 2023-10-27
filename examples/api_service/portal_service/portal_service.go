package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func main() {
	r := gin.Default()
	r.GET("/address", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})
	err := r.Run("127.0.0.1:50080")
	if err != nil {
		println(err)
	}
}
