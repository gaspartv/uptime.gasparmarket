package router

import (
	"context"
	"time"

	"github.com/gaspartv/uptime.gasparmarket/src/config"
	"github.com/gaspartv/uptime.gasparmarket/src/internal/handler"
	"github.com/gaspartv/uptime.gasparmarket/src/internal/service"
	"github.com/gin-gonic/gin"
)

func InitializeRoutes(env *config.Env) {
	router := gin.Default()

	uptimeService := service.NewUptimeService(10*time.Second, env)
	uptimeService.Start(context.Background(), env.APIChecksURL, time.Minute, 5*time.Minute)

	uptimeHandler := handler.NewUptimeHandler(uptimeService, env.APIChecksURL)
	uptimeRoutes := router.Group("uptime")
	{
		uptimeRoutes.GET("status", uptimeHandler.Status)
		uptimeRoutes.GET("check", uptimeHandler.Check)
	}

	router.Run(":" + env.Port)
}
