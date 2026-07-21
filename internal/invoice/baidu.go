package invoice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)
 
// BaiduClient 百度云 OCR 客户端，带自动 access_token 缓存
type BaiduClient struct {
	cfg        *Config
	httpClient *http.Client

	mu      sync.RWMutex
	token   string
	expires time.Time
}

// NewBaiduClient 创建百度云客户端
func NewBaiduClient(cfg *Config) *BaiduClient {
	return &BaiduClient{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// VatInvoiceResponse 百度云增值税发票识别返回值
type VatInvoiceResponse struct {
	LogID          int64                  `json:"log_id"`
	WordsResultNum int                    `json:"words_result_num"`
	WordsResult    map[string]interface{} `json:"words_result"`
	ErrorCode      int                    `json:"error_code,omitempty"`
	ErrorMessage   string                 `json:"error_msg,omitempty"`
}

// GetAccessToken 获取百度云 access_token（带缓存）
func (c *BaiduClient) GetAccessToken() (string, error) {
	c.mu.RLock()
	if c.token != "" && time.Now().Before(c.expires) {
		defer c.mu.RUnlock()
		return c.token, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// 双重检查
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}

	postData := fmt.Sprintf(
		"grant_type=client_credentials&client_id=%s&client_secret=%s",
		c.cfg.APIKey, c.cfg.SecretKey,
	)
	resp, err := c.httpClient.Post(c.cfg.TokenURL, "application/x-www-form-urlencoded", strings.NewReader(postData))
	if err != nil {
		return "", fmt.Errorf("获取百度 access_token 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取百度 token 响应失败: %w", err)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"` // 秒
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析百度 token 响应失败: %w", err)
	}
	if result.AccessToken == "" {
		return "", fmt.Errorf("百度 token 获取失败: %s", result.Error)
	}

	c.token = result.AccessToken
	// expires_in 通常为 30 天，提前 1 天刷新
	expiry := c.cfg.TokenExpiry
	if result.ExpiresIn > 0 {
		expiry = time.Duration(result.ExpiresIn-86400) * time.Second
	}
	c.expires = time.Now().Add(expiry)

	return c.token, nil
}

// isPDF 检测数据是否是 PDF 格式
func isPDF(data []byte) bool {
	return len(data) > 4 && bytes.HasPrefix(data, []byte("%PDF-"))
}

// isOFD 检测数据是否是 OFD 格式（OFD 是 ZIP 包，文件头 PK\x03\x04）
func isOFD(data []byte) bool {
	return len(data) > 4 && !isJPEG(data) && !isPNG(data) && !isBMP(data) &&
		bytes.HasPrefix(data, []byte{0x50, 0x4B, 0x03, 0x04})
}

// isJPEG 检测 JPEG 格式
func isJPEG(data []byte) bool {
	return len(data) > 2 && bytes.HasPrefix(data, []byte{0xFF, 0xD8})
}

// isPNG 检测 PNG 格式
func isPNG(data []byte) bool {
	return len(data) > 8 && bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
}

// isBMP 检测 BMP 格式
func isBMP(data []byte) bool {
	return len(data) > 2 && bytes.HasPrefix(data, []byte("BM"))
}

// getFileType 检测文件类型并返回对应的百度 API 参数名
func getFileType(data []byte) string {
	switch {
	case isPDF(data):
		return "pdf_file"
	case isOFD(data):
		return "ofd_file"
	default:
		return "image"
	}
}

// Recognize 识别增值税发票
// imageData: 图片二进制数据
// imageURL: 图片 URL（二选一，优先使用 imageData）
func (c *BaiduClient) Recognize(imageData []byte, imageURL string) (*VatInvoiceResponse, error) {
	token, err := c.GetAccessToken()
	if err != nil {
		return nil, err
	}

	apiURL := c.cfg.OCRURL + "?access_token=" + url.QueryEscape(token)

	var body io.Reader
	if len(imageData) > 0 {
		paramName := getFileType(imageData)
		fmt.Printf("[invoice] detected type: %s, sending as %s...\n", paramName, paramName)
		encoded := base64.StdEncoding.EncodeToString(imageData)
		form := url.Values{}
		form.Set(paramName, encoded)
		if paramName == "image" {
			form.Set("seal_tag", "false")
		}
		body = strings.NewReader(form.Encode())
	} else if imageURL != "" {
		form := url.Values{}
		form.Set("url", imageURL)
		form.Set("seal_tag", "false")
		body = strings.NewReader(form.Encode())
	} else {
		return nil, fmt.Errorf("请提供图片数据或图片 URL")
	}

	req, err := http.NewRequest("POST", apiURL, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("调用百度 OCR 失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取百度 OCR 响应失败: %w", err)
	}

	var result VatInvoiceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析百度 OCR 响应失败: %w", err)
	}

	if result.ErrorCode != 0 {
		return nil, fmt.Errorf("百度 OCR 返回错误 [%d]: %s", result.ErrorCode, result.ErrorMessage)
	}

	return &result, nil
}

// toString 将百度 OCR 返回的 interface{} 值转为字符串
// 兼容 string 和 float64（JSON 数字默认类型）两种类型
func toString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%.2f", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ToMap 将百度 OCR 识别结果转为 map[string]string，返回给前端字段捷径。
//
// 策略：同时输出中文 key 和英文原始 key（如 "价税合计(小写)" 和 "AmountInFiguers"），
// 前端无论用哪种命名都能匹配到。未在 fieldMap 中的字段也会透传英文 key。
func (r *VatInvoiceResponse) ToMap() map[string]string {
	m := make(map[string]string)
	if r.WordsResult == nil {
		return m
	}

	// 英文 key → 中文 key 映射表（对照百度增值税发票识别 API 文档）
	fieldMap := map[string]string{
		// 发票基本信息
		"InvoiceCode":       "发票代码",
		"InvoiceNum":        "发票号码",
		"InvoiceDate":       "开票日期",
		"InvoiceType":       "发票种类",
		"InvoiceTypeOrg":    "发票名称",
		"InvoiceTag":        "左上角标志",
		"ServiceType":       "发票消费类型",
		"MachineNum":        "机打号码",
		"MachineCode":       "机器编号",
		"CheckCode":         "校验码",
		"InvoiceNumDigit":   "数电票号",
		// 购买方信息
		"PurchaserName":        "购买方名称",
		"PurchaserRegisterNum": "购买方纳税人识别号",
		"PurchaserAddress":     "购买方地址电话",
		"PurchaserBank":        "购买方开户行账号",
		// 销售方信息
		"SellerName":        "销售方名称",
		"SellerRegisterNum": "销售方纳税人识别号",
		"SellerAddress":     "销售方地址电话",
		"SellerBank":        "销售方开户行账号",
		// 金额信息
		"TotalAmount":    "合计金额",
		"TotalTax":       "合计税额",
		"AmountInFiguers": "价税合计(小写)",
		"AmountInWords":   "价税合计(大写)",
		// 人员信息
		"Payee":      "收款人",
		"Checker":    "复核",
		"NoteDrawer": "开票人",
		// 其他信息
		"Province": "省",
		"City":     "市",
		"Agent":    "是否代开",
		"SheetNum": "联次信息",
		"Remarks":  "备注",
	}

	// 第一遍：将映射表里的字段以中文 key 输出
	for engKey, cnKey := range fieldMap {
		if val, ok := r.WordsResult[engKey]; ok {
			if s := toString(val); s != "" {
				m[cnKey] = s
			}
		}
	}

	// 第二遍：将所有百度返回的字段以英文原始 key 透传一份
	// 这样即使 fieldMap 漏了某个字段，前端仍可通过英文 key 获取
	for engKey, val := range r.WordsResult {
		if s := toString(val); s != "" {
			m[engKey] = s
		}
	}

	m["log_id"] = fmt.Sprintf("%d", r.LogID)
	return m
}
