package handler

import (
	"net/http"
	"time"

	"github.com/gaspartv/uptime.gasparmarket/src/internal/service"
	"github.com/gin-gonic/gin"
)

type UptimeHandler struct {
	service *service.UptimeService
	targets []string
}

func NewUptimeHandler(service *service.UptimeService, targets []string) *UptimeHandler {
	return &UptimeHandler{service: service, targets: targets}
}

func (h *UptimeHandler) Status(c *gin.Context) {
	results := h.service.CheckNow(c.Request.Context(), h.targets)

	status := http.StatusOK
	for _, result := range results {
		if !result.Online {
			status = http.StatusServiceUnavailable
			break
		}
	}

	c.JSON(status, gin.H{
		"checkedAt": time.Now(),
		"results":   results,
	})
}

func (h *UptimeHandler) Check(c *gin.Context) {
	h.Status(c)
}
