package handler

import (
	"bufio"
	"fmt"
	"io"
	"krillin-ai/internal/response"
	"krillin-ai/log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const cookieFilePath = "cookies.txt"

// CookieStatusResponse contains cookie file status information
type CookieStatusResponse struct {
	Exists           bool   `json:"exists"`
	LastModified     string `json:"lastModified"`
	LastModifiedTs   int64  `json:"lastModifiedTs"`
	CookieCount      int    `json:"cookieCount"`
	EarliestExpiry   string `json:"earliestExpiry"`
	EarliestExpiryTs int64  `json:"earliestExpiryTs"`
	DaysUntilExpiry  int    `json:"daysUntilExpiry"`
	Status           string `json:"status"` // "valid", "expiring_soon", "expired", "not_found"
	StatusMsg        string `json:"statusMsg"`
}

// GetCookieStatus returns the current status of the cookies.txt file
func (h Handler) GetCookieStatus(c *gin.Context) {
	log.GetLogger().Info("获取Cookie状态")

	info, err := os.Stat(cookieFilePath)
	if os.IsNotExist(err) {
		response.Success(c, CookieStatusResponse{
			Exists:    false,
			Status:    "not_found",
			StatusMsg: "Cookie文件不存在 Cookie file not found",
		})
		return
	}
	if err != nil {
		response.Error(c, 1000, "读取Cookie文件状态失败 Failed to read cookie file status")
		return
	}

	result := CookieStatusResponse{
		Exists:         true,
		LastModified:   info.ModTime().Format("2006-01-02 15:04:05"),
		LastModifiedTs: info.ModTime().Unix(),
	}

	// Parse cookies to find expiry info
	file, err := os.Open(cookieFilePath)
	if err != nil {
		response.Error(c, 1000, "打开Cookie文件失败 Failed to open cookie file")
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var earliestExpiry int64
	cookieCount := 0

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Split(line, "\t")
		if len(fields) >= 7 {
			cookieCount++
			expiryStr := fields[4]
			expiry, err := strconv.ParseInt(expiryStr, 10, 64)
			if err != nil || expiry == 0 {
				continue // Skip session cookies (expiry=0)
			}
			if earliestExpiry == 0 || expiry < earliestExpiry {
				earliestExpiry = expiry
			}
		}
	}

	result.CookieCount = cookieCount

	if earliestExpiry > 0 {
		expiryTime := time.Unix(earliestExpiry, 0)
		result.EarliestExpiry = expiryTime.Format("2006-01-02 15:04:05")
		result.EarliestExpiryTs = earliestExpiry
		daysUntil := int(time.Until(expiryTime).Hours() / 24)
		result.DaysUntilExpiry = daysUntil

		if daysUntil < 0 {
			result.Status = "expired"
			result.StatusMsg = fmt.Sprintf("Cookie已过期%d天 Cookie expired %d days ago", -daysUntil, -daysUntil)
		} else if daysUntil < 7 {
			result.Status = "expiring_soon"
			result.StatusMsg = fmt.Sprintf("Cookie将在%d天后过期 Cookie expires in %d days", daysUntil, daysUntil)
		} else {
			result.Status = "valid"
			result.StatusMsg = fmt.Sprintf("Cookie有效，%d天后过期 Cookie valid, expires in %d days", daysUntil, daysUntil)
		}
	} else {
		// All session cookies, check file age
		daysSinceModified := int(time.Since(info.ModTime()).Hours() / 24)
		if daysSinceModified > 30 {
			result.Status = "expiring_soon"
			result.StatusMsg = fmt.Sprintf("Cookie文件已%d天未更新，建议更新 Cookie file not updated for %d days", daysSinceModified, daysSinceModified)
		} else {
			result.Status = "valid"
			result.StatusMsg = "Cookie文件存在 Cookie file exists"
		}
		result.DaysUntilExpiry = -1 // Unknown
	}

	response.Success(c, result)
}

// UploadCookie handles cookie content upload (text paste or file upload)
func (h Handler) UploadCookie(c *gin.Context) {
	log.GetLogger().Info("上传Cookie文件")

	var cookieContent string

	// Try to get from multipart file first
	file, _, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		data, err := io.ReadAll(file)
		if err != nil {
			response.Error(c, 1001, "读取上传文件失败 Failed to read uploaded file")
			return
		}
		cookieContent = string(data)
	} else {
		// Try to get from JSON body or form field
		type UploadRequest struct {
			Content string `json:"content" form:"content"`
		}
		var req UploadRequest
		if err := c.ShouldBind(&req); err != nil || req.Content == "" {
			response.Error(c, 1001, "请提供Cookie内容 Please provide cookie content")
			return
		}
		cookieContent = req.Content
	}

	// Basic validation: check Netscape format
	lines := strings.Split(cookieContent, "\n")
	validCookieLines := 0
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) >= 7 {
			validCookieLines++
		}
	}

	if validCookieLines == 0 {
		response.Error(c, 1001, "无效的Cookie格式，请使用Netscape格式 Invalid cookie format, please use Netscape format")
		return
	}

	// Write to cookies.txt
	err = os.WriteFile(cookieFilePath, []byte(cookieContent), 0644)
	if err != nil {
		log.GetLogger().Error("写入Cookie文件失败", zap.Error(err))
		response.Error(c, 1502, "写入Cookie文件失败 Failed to write cookie file")
		return
	}

	log.GetLogger().Info("Cookie文件更新成功", zap.Int("validCookies", validCookieLines))
	response.Success(c, gin.H{
		"cookieCount": validCookieLines,
		"message":     fmt.Sprintf("成功保存%d条Cookie Successfully saved %d cookies", validCookieLines, validCookieLines),
	})
}

// ValidateCookie tests if the current cookies work with yt-dlp
func (h Handler) ValidateCookie(c *gin.Context) {
	log.GetLogger().Info("验证Cookie有效性")

	_, err := os.Stat(cookieFilePath)
	if os.IsNotExist(err) {
		response.Error(c, 1104, "Cookie文件不存在 Cookie file not found")
		return
	}

	// Use yt-dlp to test cookies by fetching info from a known public video
	cmd := exec.Command("/app/bin/yt-dlp", "--cookies", cookieFilePath,
		"--dump-json", "--no-download",
		"https://www.youtube.com/watch?v=dQw4w9WgXcQ")
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if err != nil {
		if strings.Contains(outputStr, "Sign in to confirm") || strings.Contains(outputStr, "LOGIN_REQUIRED") {
			response.Error(c, 1104, "Cookie已过期或无效，请重新导出 Cookie expired or invalid, please re-export")
			return
		}
		response.Error(c, 1104, fmt.Sprintf("Cookie验证失败 Cookie validation failed: %s", truncateString(outputStr, 200)))
		return
	}

	response.Success(c, gin.H{
		"valid":   true,
		"message": "Cookie验证通过 Cookie validation passed",
	})
}

// truncateString truncates a string to maxLen characters
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
