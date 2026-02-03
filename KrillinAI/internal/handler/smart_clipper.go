package handler

import (
	"krillin-ai/internal/dto"
	"krillin-ai/internal/service"
	"krillin-ai/log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h Handler) AnalyzeVideo(c *gin.Context) {
	var req dto.SmartClipperAnalyzeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.GetLogger().Error("AnalyzeVideo param error", zap.Error(err))
		c.JSON(http.StatusOK, dto.SmartClipperAnalyzeRes{
			Error: 1,
			Msg:   "params error: " + err.Error(),
		})
		return
	}

	svc := service.NewService()
	data, err := svc.AnalyzeVideo(req)
	if err != nil {
		log.GetLogger().Error("AnalyzeVideo failed", zap.Error(err))
		c.JSON(http.StatusOK, dto.SmartClipperAnalyzeRes{
			Error: 500,
			Msg:   "Analysis failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SmartClipperAnalyzeRes{
		Error: 0,
		Msg:   "success",
		Data:  data,
	})
}

func (h Handler) SubmitClips(c *gin.Context) {
	var req dto.SmartClipperSubmitReq
	if err := c.ShouldBindJSON(&req); err != nil {
		log.GetLogger().Error("SubmitClips param error", zap.Error(err))
		c.JSON(http.StatusOK, dto.SmartClipperSubmitRes{
			Error: 1,
			Msg:   "params error: " + err.Error(),
		})
		return
	}

	svc := service.NewService()
	data, err := svc.SubmitClips(req)
	if err != nil {
		log.GetLogger().Error("SubmitClips failed", zap.Error(err))
		c.JSON(http.StatusOK, dto.SmartClipperSubmitRes{
			Error: 500,
			Msg:   "Submission failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, dto.SmartClipperSubmitRes{
		Error: 0,
		Msg:   "success",
		Data:  data,
	})
}
