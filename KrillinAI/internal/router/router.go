package router

import (
	"krillin-ai/internal/handler"
	"krillin-ai/log"
	"krillin-ai/static"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func SetupRouter(r *gin.Engine) {
	api := r.Group("/api")

	hdl := handler.NewHandler()
	{
		api.POST("/capability/subtitleTask", hdl.StartSubtitleTask)
		api.GET("/capability/subtitleTask", hdl.GetSubtitleTask)
		api.GET("/capability/history", hdl.GetTaskHistory)      // New History API
		api.DELETE("/capability/task/:taskId", hdl.DeleteTask)  // New Delete API
		api.POST("/capability/task/:taskId/retry", hdl.RetryTask) // Retry Failed Task
		api.POST("/file", hdl.UploadFile)
		api.GET("/file/*filepath", hdl.DownloadFile)
		api.HEAD("/file/*filepath", hdl.DownloadFile)
		api.GET("/config", hdl.GetConfig)
		api.POST("/config", hdl.UpdateConfig)
		// Smart Clipper Routes
		api.POST("/smart_clipper/analyze", hdl.AnalyzeVideo)
		api.POST("/smart_clipper/submit", hdl.SubmitClips)
		// Cookie Management Routes
		api.GET("/cookie/status", hdl.GetCookieStatus)
		api.POST("/cookie/upload", hdl.UploadCookie)
		api.POST("/cookie/validate", hdl.ValidateCookie)
	}

	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/static")
	})
	if _, err := os.Stat("static"); err == nil {
		log.GetLogger().Info("Using local static directory")
		r.Static("/static", "static")
	} else {
		log.GetLogger().Info("Using embedded static files")
		r.StaticFS("/static", http.FS(static.EmbeddedFiles))
	}
}
