package invoice

import (
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
	WordsResult    map[string]WordUnit `json:"words_result"`
	ErrorCode      int                    `json:"error_code,omitempty"`
	ErrorMessage   string                 `json:"error_msg,omitempty"`
}

type WordUnit struct {
	Word string `json:"word"`
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
		// base64 编码传输
		encoded := base64.StdEncoding.EncodeToString(imageData)
		form := url.Values{}
		form.Set("image", encoded)
		form.Set("seal_tag", "false") // 不检测印章
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

// ToMap 将识别结果转为 map[string]string，方便返回给前端
func (r *VatInvoiceResponse) ToMap() map[string]string {
	m := make(map[string]string)
	if r.WordsResult == nil {
		return m
	}

	fieldMap := map[string]string{
		"InvoiceCode":          "发票代码",
		"InvoiceNum":           "发票号码",
		"InvoiceDate":          "开票日期",
		"PurchaserName":        "购买方名称",
		"PurchaserRegisterNum": "购买方纳税人识别号",
		"PurchaserAddress":     "购买方地址电话",
		"PurchaserBank":        "购买方开户行账号",
		"SellerName":           "销售方名称",
		"SellerRegisterNum":    "销售方纳税人识别号",
		"SellerAddress":        "销售方地址电话",
		"SellerBank":           "销售方开户行账号",
		"TotalAmount":          "合计金额",
		"TotalTax":             "合计税额",
		"TotalPrice":           "合计总额",
		"TotalPriceWords":      "合计总额大写",
		"CheckCode":            "校验码",
		"Payee":                "收款人",
		"Checker":              "复核人",
		"Note":                 "备注",
		"InvoiceType":          "发票种类",
		"Province":             "省份",
		"City":                 "城市",
		"SheetNum":             "所属联次",
		"Password":             "密码区",
		"OnlinePay":            "在线支付",
		"ServiceName":          "服务名称",
		"ServiceType":          "服务类型",
	}

	for engKey, cnKey := range fieldMap {
		if unit, ok := r.WordsResult[engKey]; ok && unit.Word != "" {
			m[cnKey] = unit.Word
		}
	}

	m["log_id"] = fmt.Sprintf("%d", r.LogID)
	return m
}
