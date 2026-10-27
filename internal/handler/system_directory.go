package handler

import (
	"net/http"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/gin-gonic/gin"
)

// GetDirectorySync returns the latest sync run summary.
func (h *SystemHandler) GetDirectorySync(c *gin.Context) {
	if h.directorySync == nil {
		c.JSON(http.StatusOK, gin.H{"status": "idle"})
		return
	}
	state, err := h.directorySync.State(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "服务器繁忙，请稍后重试"})
		return
	}
	c.JSON(http.StatusOK, state)
}

// PostDirectorySync triggers a directory sync. Query full=1 for a full resync.
func (h *SystemHandler) PostDirectorySync(c *gin.Context) {
	if h.directorySync == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "message": "目录同步未配置"})
		return
	}
	full := c.Query("full") == "1"
	actorID, _ := types.UserIDFromContext(c.Request.Context())
	state, err := h.directorySync.Sync(c.Request.Context(), actorID, full)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, state)
}
