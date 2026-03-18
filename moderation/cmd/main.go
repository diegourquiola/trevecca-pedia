package main

import (
	"moderation/config"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.SetTrustedProxies(nil)
	gin.SetMode(gin.DebugMode)

	port := config.GetEnv("MOD_SERVICE_PORT", "6633")
	r.Run(":" + port)
}
