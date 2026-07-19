package invoice

import (
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/SpadeGo/feishu-extension-services/internal/core"
	"github.com/gin-gonic/gin"
)

// OCRResponse 前端返回结构
type OCRResponse struct {
	LogID       int64             `json:"log_id"`
	WordsResult map[string]string `json:"words_result"`
}

type OCRRequest struct {
	ImageBase64 string `json:"image_base64"` // base64 编码的图片
	ImageURL    string `json:"image_url"`    // 图片 URL（二选一）
}

const (
	CodeBadRequest  = 20001
	CodeOCRFailed   = 20002
	CodeTokenFailed = 20003
	CodeNoImage     = 20004
)

// Handler 发票识别 Handler
type Handler struct {
	baidu *BaiduClient
}

func NewHandler(cfg *Config) *Handler {
	return &Handler{
		baidu: NewBaiduClient(cfg),
	}
}

func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.POST("/invoice/ocr", h.ocr)
	rg.POST("/invoice/ocr-file", h.ocrFile)
}

// ocr 接收 JSON（base64 图片或图片 URL）
func (h *Handler) ocr(c *gin.Context) {
	var req OCRRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		core.Fail(c, CodeBadRequest, "请求参数错误: "+err.Error())
		return
	}

	if req.ImageBase64 == "" && req.ImageURL == "" {
		core.Fail(c, CodeNoImage, "请提供 image_base64 或 image_url")
		return
	}

	var result *VatInvoiceResponse
	var err error

	if req.ImageBase64 != "" {
		imageData, decodeErr := base64.StdEncoding.DecodeString(req.ImageBase64)
		if decodeErr != nil {
			core.Fail(c, CodeBadRequest, "图片 base64 解码失败: "+decodeErr.Error())
			return
		}
		result, err = h.baidu.Recognize(imageData, "")
	} else {
		result, err = h.baidu.Recognize(nil, req.ImageURL)
	}

	if err != nil {
		core.Fail(c, CodeOCRFailed, "发票识别失败: "+err.Error())
		return
	}

	core.Success(c, OCRResponse{
		LogID:       result.LogID,
		WordsResult: result.ToMap(),
	})
}

// ocrFile 接收 multipart 文件上传
func (h *Handler) ocrFile(c *gin.Context) {
	file, err := c.FormFile("image")
	if err != nil {
		core.Fail(c, CodeBadRequest, "请上传图片文件")
		return
	}

	if file.Size > 10*1024*1024 {
		core.Fail(c, CodeBadRequest, "图片文件不能超过 10MB")
		return
	}

	src, err := file.Open()
	if err != nil {
		core.Fail(c, CodeBadRequest, "读取文件失败")
		return
	}
	defer src.Close()

	imageData, err := io.ReadAll(src)
	if err != nil {
		core.Fail(c, CodeBadRequest, "读取文件数据失败")
		return
	}

	result, err := h.baidu.Recognize(imageData, "")
	if err != nil {
		core.Fail(c, CodeOCRFailed, "发票识别失败: "+err.Error())
		return
	}

	core.Success(c, OCRResponse{
		LogID:       result.LogID,
		WordsResult: result.ToMap(),
	})
}

// 确保导入不被优化掉
var _ = json.Marshal
