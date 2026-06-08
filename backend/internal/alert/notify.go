package alert

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

// 东八区，用于告警时间展示（避免 UTC 与本地差 8 小时）
var locShanghai *time.Location

func init() {
	locShanghai, _ = time.LoadLocation("Asia/Shanghai")
	if locShanghai == nil {
		locShanghai = time.FixedZone("CST", 8*3600)
	}
}

// FormatTimeLocal 将时间格式化为东八区 "2006-01-02 15:04:05"，供模板 {{.Time}} 使用
func FormatTimeLocal(t time.Time) string {
	if locShanghai != nil {
		t = t.In(locShanghai)
	}
	return t.Format("2006-01-02 15:04:05")
}

// SendTestNotification 发送测试通知
func SendTestNotification(channel *NotifyChannel) error {
	testMsg := NotifyMessage{
		Title:    "【测试通知】Ops Platform 告警中心",
		RuleName: "系统连通性测试",
		Content:  "这是一条测试通知消息，用于验证通知渠道是否配置正确。",
		Severity: "info",
		Status:   "testing",
		Source:   "系统测试",
		Time:     FormatTimeLocal(time.Now()),
		Category: "system",
	}
	return SendNotification(channel, &testMsg)
}

// NotifyMessage 通知消息体
//
// 告警模板字段说明：
//   - Title:       通知标题，如 "【告警】磁盘使用率过高"
//   - RuleName:    告警规则名称，如 "DiskUsageHigh"
//   - Content:     告警详细内容/描述
//   - CurrentValue: 当前指标值（恢复告警时查询 Prometheus 获取）
//   - Severity:    告警级别 (critical=严重, warning=警告, info=提醒)
//   - Status:      告警状态 (firing=告警中, resolved=已恢复, testing=测试)
//   - Source:      告警来源，通常为实例 IP:Port
//   - Category:    告警分类 (disk/memory/cpu/instance/network/load/other)
//   - Time:        触发/恢复时间
type NotifyMessage struct {
	Title       string
	RuleName    string
	Content     string
	CurrentValue string
	Severity    string
	Status      string
	Source      string
	Category    string
	Time        string
}

// severityMap 告警级别中文映射
var severityMap = map[string]string{
	"critical": "严重",
	"warning":  "警告",
	"info":     "提醒",
}

// statusMap 告警状态中文映射
var statusMap = map[string]string{
	"firing":       "告警中",
	"resolved":     "已恢复",
	"acknowledged": "已确认",
	"closed":       "已关闭",
	"testing":      "测试",
}

// categoryMap 告警分类中文映射
var categoryMap = map[string]string{
	"disk":     "磁盘",
	"memory":   "内存",
	"cpu":      "CPU",
	"instance": "实例存活",
	"network":  "网络",
	"load":     "负载",
	"system":   "系统",
	"other":    "其他",
}

// getSeverityLabel 获取级别中文标签
func getSeverityLabel(s string) string {
	if v, ok := severityMap[s]; ok {
		return v
	}
	return s
}

// getStatusLabel 获取状态中文标签
func getStatusLabel(s string) string {
	if v, ok := statusMap[s]; ok {
		return v
	}
	return s
}

// getCategoryLabel 获取分类中文标签
func getCategoryLabel(s string) string {
	if v, ok := categoryMap[s]; ok {
		return v
	}
	return s
}

// getSeverityEmoji 获取级别 Emoji
func getSeverityEmoji(severity, status string) string {
	if status == "resolved" {
		return "✅"
	}
	if status == "testing" {
		return "🔔"
	}
	switch severity {
	case "critical":
		return "🔴"
	case "warning":
		return "🟡"
	case "info":
		return "🔵"
	default:
		return "⚪"
	}
}

// SendNotification 发送通知
// 优先使用数据库中配置的模板，如果没有配置则使用硬编码的默认模板
func SendNotification(channel *NotifyChannel, msg *NotifyMessage) error {
	// 确定场景
	scene := msg.Status
	if scene == "testing" {
		scene = "firing" // 测试使用触发模板
	}
	if scene != "firing" && scene != "resolved" {
		scene = "firing"
	}

	// 查找匹配的模板
	tpl := getTemplateForChannel(channel.Type, scene)

	if tpl != nil {
		// 使用数据库模板发送
		return sendWithTemplate(channel, msg, tpl)
	}

	// 回退到硬编码模板
	switch channel.Type {
	case "dingtalk":
		return sendDingTalk(channel, msg)
	case "wechat":
		return sendWeChatWork(channel, msg)
	case "email":
		return sendEmail(channel, msg)
	default:
		return fmt.Errorf("不支持的通知类型: %s", channel.Type)
	}
}

// sendWithTemplate 使用数据库模板发送通知
func sendWithTemplate(channel *NotifyChannel, msg *NotifyMessage, tpl *AlertTemplate) error {
	data := buildTemplateData(msg)

	log.Printf("[alert notify] 模板渲染数据: Content=%s", data.Content)

	// 渲染标题
	title := msg.Title
	if tpl.TitleTpl != "" {
		if rendered, err := renderTemplate(tpl.TitleTpl, data); err == nil {
			title = rendered
		} else {
			log.Printf("[alert notify] 标题模板渲染失败: %v", err)
		}
	}

	// 渲染内容
	content, err := renderTemplate(tpl.ContentTpl, data)
	if err != nil {
		log.Printf("[alert notify] 内容模板渲染失败，使用原始内容: %v", err)
		// 渲染失败，回退到原始内容
		content = msg.Content
	}

	log.Printf("[alert notify] 最终发送内容: %s", content)

	switch channel.Type {
	case "dingtalk":
		return sendDingTalkWithContent(channel, title, content)
	case "wechat":
		return sendWeChatWithContent(channel, title, content)
	case "email":
		return sendEmailWithContent(channel, title, content, msg.Severity)
	default:
		return fmt.Errorf("不支持的通知类型: %s", channel.Type)
	}
}

// sendDingTalkWithContent 使用已渲染内容发送钉钉消息
func sendDingTalkWithContent(channel *NotifyChannel, title, content string) error {
	webhookURL := strings.TrimSpace(channel.WebhookURL)
	secret := strings.TrimSpace(channel.Secret)

	if secret != "" {
		timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
		stringToSign := fmt.Sprintf("%s\n%s", timestamp, secret)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s", webhookURL, timestamp, sign)
	}

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  content,
		},
	}

	return postJSON(webhookURL, payload)
}

// sendWeChatWithContent 使用已渲染内容发送企微消息
func sendWeChatWithContent(channel *NotifyChannel, title string, content string) error {
	_ = title // 企微 markdown 没有单独的 title 字段
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
	return postJSON(channel.WebhookURL, payload)
}

// sendEmailWithContent 使用已渲染内容发送邮件
func sendEmailWithContent(channel *NotifyChannel, title, content, severity string) error {
	if channel.SMTPHost == "" || channel.SMTPUser == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}

	from := channel.EmailFrom
	if from == "" {
		from = channel.SMTPUser
	}

	sevColor := map[string]string{"critical": "#f5222d", "warning": "#faad14", "info": "#1890ff"}
	color := sevColor[severity]
	if color == "" {
		color = "#666"
	}

	body := fmt.Sprintf(
		`<html><body style="font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',Roboto,sans-serif;padding:20px;background:#f5f5f5;">
<div style="max-width:600px;margin:0 auto;background:#fff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.1);">
  <div style="background:%s;color:#fff;padding:16px 24px;">
    <h2 style="margin:0;font-size:18px;">%s</h2>
  </div>
  <div style="padding:24px;">%s</div>
  <div style="padding:16px 24px;background:#fafafa;color:#999;font-size:12px;text-align:center;">
    Ops Platform 告警中心 · 自动发送，请勿直接回复
  </div>
</div>
</body></html>`, color, title, content)

	header := make(map[string]string)
	header["From"] = from
	header["Subject"] = fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(title)))
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range header {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	addr := fmt.Sprintf("%s:%d", channel.SMTPHost, channel.SMTPPort)
	auth := smtp.PlainAuth("", channel.SMTPUser, channel.SMTPPass, channel.SMTPHost)

	return smtp.SendMail(addr, auth, from, []string{from}, []byte(message.String()))
}

// sendDingTalk 发送钉钉机器人消息
//
// 钉钉告警模板格式：
//   ### 🔴 【告警】磁盘使用率过高
//   > **规则名称**：DiskUsageHigh
//   > **告警内容**：/dev/sda1 磁盘使用率达到 95%
//   > **来源**：10.99.99.100:9100
//   > **级别**：严重
//   > **分类**：磁盘
//   > **状态**：告警中
//   > **触发时间**：2026-02-09 12:00:00
func sendDingTalk(channel *NotifyChannel, msg *NotifyMessage) error {
	webhookURL := strings.TrimSpace(channel.WebhookURL)
	secret := strings.TrimSpace(channel.Secret)

	// 如果配置了签名密钥，计算签名
	if secret != "" {
		timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
		stringToSign := fmt.Sprintf("%s\n%s", timestamp, secret)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s", webhookURL, timestamp, sign)
	}

	emoji := getSeverityEmoji(msg.Severity, msg.Status)

	// Markdown 消息格式
	markdown := fmt.Sprintf("### %s %s\n\n"+
		"> **规则名称**：%s\n\n"+
		"> **告警内容**：%s\n\n"+
		"> **来源**：%s\n\n"+
		"> **级别**：%s\n\n"+
		"> **分类**：%s\n\n"+
		"> **状态**：%s\n\n"+
		"> **触发时间**：%s\n",
		emoji, msg.Title,
		msg.RuleName,
		msg.Content,
		msg.Source,
		getSeverityLabel(msg.Severity),
		getCategoryLabel(msg.Category),
		getStatusLabel(msg.Status),
		msg.Time,
	)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": msg.Title,
			"text":  markdown,
		},
	}

	return postJSON(webhookURL, payload)
}

// sendWeChatWork 发送企业微信机器人消息
//
// 企微告警模板格式（与钉钉类似，使用企微 Markdown 语法）
func sendWeChatWork(channel *NotifyChannel, msg *NotifyMessage) error {
	// 企微颜色：info=绿色, comment=灰色, warning=橙色
	severityColor := map[string]string{"critical": "warning", "warning": "warning", "info": "info"}
	color := severityColor[msg.Severity]
	if color == "" {
		color = "comment"
	}

	statusColor := "warning"
	if msg.Status == "resolved" {
		statusColor = "info"
	}

	content := fmt.Sprintf("## %s\n"+
		"> **规则名称**：%s\n"+
		"> **告警内容**：%s\n"+
		"> **来源**：%s\n"+
		"> **级别**：<font color=\"%s\">%s</font>\n"+
		"> **分类**：%s\n"+
		"> **状态**：<font color=\"%s\">%s</font>\n"+
		"> **触发时间**：%s",
		msg.Title,
		msg.RuleName,
		msg.Content,
		msg.Source,
		color, getSeverityLabel(msg.Severity),
		getCategoryLabel(msg.Category),
		statusColor, getStatusLabel(msg.Status),
		msg.Time,
	)

	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}

	return postJSON(channel.WebhookURL, payload)
}

// sendEmail 发送邮件通知
//
// 邮件告警模板：使用 HTML 表格展示所有字段，附带级别颜色标识
func sendEmail(channel *NotifyChannel, msg *NotifyMessage) error {
	if channel.SMTPHost == "" || channel.SMTPUser == "" {
		return fmt.Errorf("SMTP 配置不完整")
	}

	from := channel.EmailFrom
	if from == "" {
		from = channel.SMTPUser
	}

	sevLabel := getSeverityLabel(msg.Severity)
	statLabel := getStatusLabel(msg.Status)
	catLabel := getCategoryLabel(msg.Category)

	// 级别对应的颜色
	sevColor := map[string]string{"critical": "#f5222d", "warning": "#faad14", "info": "#1890ff"}
	color := sevColor[msg.Severity]
	if color == "" {
		color = "#666"
	}

	// 状态对应的颜色
	statColor := "#faad14"
	if msg.Status == "resolved" {
		statColor = "#52c41a"
	}

	subject := fmt.Sprintf("[%s] %s", sevLabel, msg.Title)
	body := fmt.Sprintf(
		`<html><body style="font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 20px; background: #f5f5f5;">
<div style="max-width: 600px; margin: 0 auto; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 2px 8px rgba(0,0,0,0.1);">
  <div style="background: %s; color: #fff; padding: 16px 24px;">
    <h2 style="margin: 0; font-size: 18px;">%s</h2>
  </div>
  <div style="padding: 24px;">
    <table style="width: 100%%; border-collapse: collapse; font-size: 14px;">
      <tr style="border-bottom: 1px solid #f0f0f0;">
        <td style="padding: 12px 8px; color: #666; width: 100px; font-weight: bold;">规则名称</td>
        <td style="padding: 12px 8px;">%s</td>
      </tr>
      <tr style="border-bottom: 1px solid #f0f0f0;">
        <td style="padding: 12px 8px; color: #666; font-weight: bold;">告警内容</td>
        <td style="padding: 12px 8px;">%s</td>
      </tr>
      <tr style="border-bottom: 1px solid #f0f0f0;">
        <td style="padding: 12px 8px; color: #666; font-weight: bold;">来源</td>
        <td style="padding: 12px 8px; font-family: monospace;">%s</td>
      </tr>
      <tr style="border-bottom: 1px solid #f0f0f0;">
        <td style="padding: 12px 8px; color: #666; font-weight: bold;">级别</td>
        <td style="padding: 12px 8px;"><span style="background: %s; color: #fff; padding: 2px 10px; border-radius: 4px; font-size: 12px;">%s</span></td>
      </tr>
      <tr style="border-bottom: 1px solid #f0f0f0;">
        <td style="padding: 12px 8px; color: #666; font-weight: bold;">分类</td>
        <td style="padding: 12px 8px;">%s</td>
      </tr>
      <tr style="border-bottom: 1px solid #f0f0f0;">
        <td style="padding: 12px 8px; color: #666; font-weight: bold;">状态</td>
        <td style="padding: 12px 8px;"><span style="background: %s; color: #fff; padding: 2px 10px; border-radius: 4px; font-size: 12px;">%s</span></td>
      </tr>
      <tr>
        <td style="padding: 12px 8px; color: #666; font-weight: bold;">触发时间</td>
        <td style="padding: 12px 8px;">%s</td>
      </tr>
    </table>
  </div>
  <div style="padding: 16px 24px; background: #fafafa; color: #999; font-size: 12px; text-align: center;">
    Ops Platform 告警中心 · 自动发送，请勿直接回复
  </div>
</div>
</body></html>`,
		color, msg.Title,
		msg.RuleName,
		msg.Content,
		msg.Source,
		color, sevLabel,
		catLabel,
		statColor, statLabel,
		msg.Time,
	)

	// 构建邮件内容
	header := make(map[string]string)
	header["From"] = from
	header["Subject"] = fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(subject)))
	header["MIME-Version"] = "1.0"
	header["Content-Type"] = "text/html; charset=UTF-8"

	var message strings.Builder
	for k, v := range header {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	message.WriteString("\r\n")
	message.WriteString(body)

	addr := fmt.Sprintf("%s:%d", channel.SMTPHost, channel.SMTPPort)
	auth := smtp.PlainAuth("", channel.SMTPUser, channel.SMTPPass, channel.SMTPHost)

	// 邮件先发给联系人的关联报警组成员（这里简化为发给 SMTPUser 自己做测试）
	return smtp.SendMail(addr, auth, from, []string{from}, []byte(message.String()))
}

// postJSON 发送 JSON POST 请求，并检查响应体中的错误信息
func postJSON(targetURL string, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("序列化失败: %v", err)
	}

	resp, err := http.Post(targetURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body := new(bytes.Buffer)
	body.ReadFrom(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 状态码: %d, 响应: %s", resp.StatusCode, body.String())
	}

	// 解析响应 JSON，检查钉钉/企微的业务错误码
	// 钉钉返回: {"errcode": 0, "errmsg": "ok"} 表示成功
	// 企微返回: {"errcode": 0, "errmsg": "ok"} 表示成功
	var result struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.Unmarshal(body.Bytes(), &result); err == nil {
		if result.ErrCode != 0 {
			return fmt.Errorf("发送失败 (errcode=%d): %s", result.ErrCode, result.ErrMsg)
		}
	}

	return nil
}
