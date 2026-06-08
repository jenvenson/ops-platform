package jenkins

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// Client Jenkins客户端结构体
type Client struct {
	BaseURL  string
	Username string
	Password string
	Client   *http.Client
	Cookies  []*http.Cookie // 保存session cookies
}

type ScriptApprovalResult struct {
	ApprovedScripts    int
	ApprovedSignatures int
}

func (r ScriptApprovalResult) ApprovedCount() int {
	return r.ApprovedScripts + r.ApprovedSignatures
}

// BuildParams 构建参数
type BuildParams struct {
	APP   string `json:"app"`   // Jenkins 参数名为小写
	TAG   string `json:"tag"`   // Jenkins 参数名为小写
	SCOPE string `json:"scope"` // Jenkins 参数名为小写
}

// BuildStatus 构建状态
type BuildStatus struct {
	Phase             string `json:"phase"`
	Result            string `json:"result"`
	URL               string `json:"url"`
	Duration          int64  `json:"duration"`          // 毫秒
	EstimatedDuration int64  `json:"estimatedDuration"` // 预估持续时间（毫秒）
	Timestamp         string `json:"timestamp"`
	BuildTimestamp    int64  `json:"buildTimestamp"` // 构建开始时间戳（毫秒）
}

// QueueItem 队列项
type QueueItem struct {
	ID         int64 `json:"id"`
	Executable *struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	} `json:"executable"`
}

// NewClient 创建Jenkins客户端（支持可选 timeout 参数）
func NewClient(baseURL, username, password string, timeout ...time.Duration) *Client {
	t := 60 * time.Second
	if len(timeout) > 0 {
		t = timeout[0]
	}
	return &Client{
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		Username: username,
		Password: password,
		Client: &http.Client{
			Timeout: t,
		},
	}
}

// NewClientWithTimeout 创建带超时的Jenkins客户端
func NewClientWithTimeout(baseURL, username, password string, timeout time.Duration) *Client {
	return &Client{
		BaseURL:  strings.TrimSuffix(baseURL, "/"),
		Username: username,
		Password: password,
		Client: &http.Client{
			Timeout: timeout,
		},
	}
}

// GetCrumb 获取CSRF保护令牌（保留向后兼容）
func (c *Client) GetCrumb() (string, string, error) {
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		return "", "", err
	}
	// 保存到 c.Cookies 以保持向后兼容
	c.Cookies = cookies
	return crumb, crumbField, nil
}

// GetCrumbWithCookies 获取CSRF保护令牌和对应的session cookies
// 返回：crumb值, crumb字段名, session cookies, 错误
func (c *Client) GetCrumbWithCookies() (string, string, []*http.Cookie, error) {
	req, err := http.NewRequest("GET", c.BaseURL+"/crumbIssuer/api/json", nil)
	if err != nil {
		return "", "", nil, err
	}

	// 使用API Token进行认证
	req.SetBasicAuth(c.Username, c.Password)

	resp, err := c.Client.Do(req)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get crumb: %v", err)
	}
	defer resp.Body.Close()

	// 获取session cookies（关键：返回给调用者使用）
	cookies := resp.Cookies()
	if len(cookies) > 0 {
		fmt.Printf("[Jenkins] 获取了 %d 个 session cookies\n", len(cookies))
	}

	if resp.StatusCode == 404 {
		// Jenkins可能没有启用CSRF保护
		return "", "", nil, fmt.Errorf("CSRF protection not enabled (404)")
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", nil, fmt.Errorf("crumb request failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var crumbResp struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&crumbResp); err != nil {
		return "", "", nil, fmt.Errorf("failed to decode crumb response: %v", err)
	}

	// 验证 crumb 不为空
	if crumbResp.Crumb == "" {
		return "", "", nil, fmt.Errorf("received empty crumb from Jenkins")
	}

	return crumbResp.Crumb, crumbResp.CrumbRequestField, cookies, nil
}

// basicAuth returns the base64 encoded username:password for basic auth
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// BuildJobWithParams 触发带参数的Jenkins构建
func (c *Client) BuildJobWithParams(jobPath string, params map[string]string) (int64, error) {
	fmt.Printf("[Jenkins] 开始触发构建，jobPath: %s\n", jobPath)
	fmt.Printf("[Jenkins] 构建参数: %+v\n", params)

	// 获取CSRF令牌和对应的cookies
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		// 如果获取crumb失败，返回错误（Jenkins必须启用CSRF保护）
		return 0, fmt.Errorf("获取Jenkins crumb失败，请检查Jenkins配置: %v", err)
	}

	fmt.Printf("[Jenkins] 成功获取crumb: %s, field: %s\n", crumb, crumbField)

	// 第一次尝试
	queueID, err := c.buildWithCrumb(jobPath, params, crumb, crumbField, cookies)
	if err == nil {
		return queueID, nil
	}

	// 如果是403错误，尝试重新获取crumb并重试
	if strings.Contains(err.Error(), "403") || strings.Contains(err.Error(), "crumb") {
		fmt.Printf("[Jenkins] 第一次请求失败，尝试重新获取crumb: %v\n", err)

		// 等待一小段时间再重试
		time.Sleep(100 * time.Millisecond)

		crumb, crumbField, cookies, err = c.GetCrumbWithCookies()
		if err != nil {
			return 0, fmt.Errorf("重新获取crumb失败: %v", err)
		}

		fmt.Printf("[Jenkins] 重新获取crumb成功: %s, field: %s\n", crumb, crumbField)

		// 第二次尝试
		queueID, err = c.buildWithCrumb(jobPath, params, crumb, crumbField, cookies)
		if err != nil {
			return 0, fmt.Errorf("重试后仍然失败: %v", err)
		}

		return queueID, nil
	}

	return 0, err
}

// buildWithCrumb 使用指定的crumb和cookies执行构建
func (c *Client) buildWithCrumb(jobPath string, params map[string]string, crumb, crumbField string, cookies []*http.Cookie) (int64, error) {
	// 构建参数化请求
	formData := url.Values{}
	for key, value := range params {
		formData.Set(key, value)
	}

	// 将crumb添加到表单数据中
	if crumbField != "" {
		formData.Set(crumbField, crumb)
	} else {
		formData.Set("Jenkins-Crumb", crumb)
	}
	fmt.Printf("[Jenkins] crumb已添加到form data, field: %s\n", crumbField)

	// 创建构建请求
	jobPath = strings.TrimPrefix(jobPath, "/")
	jobPath = strings.TrimSuffix(jobPath, "/")

	// 处理不同类型的 job 路径
	// 1. 如果以 view/ 开头，不添加 job/ 前缀（例如：view/auto-archive-deploy/job/fscr-aggregation）
	// 2. 如果已经包含 /job/，不添加前缀（例如：auto-archive-deploy/job/fscr-aggregation）
	// 3. 否则添加 job/ 前缀（例如：fscr-aggregation -> job/fscr-aggregation）
	if strings.HasPrefix(jobPath, "view/") || strings.Contains(jobPath, "/job/") {
		// 路径已经正确，不需要添加前缀
	} else if !strings.HasPrefix(jobPath, "job/") {
		jobPath = "job/" + jobPath
	}

	reqURL := fmt.Sprintf("%s/%s/buildWithParameters", strings.TrimSuffix(c.BaseURL, "/"), jobPath)

	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return 0, err
	}

	// 使用API Token进行认证
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 添加对应的 session cookies（与 crumb 来自同一个请求）
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		fmt.Printf("[Jenkins] 已添加 %d 个 session cookies\n", len(cookies))
	}

	// 将crumb添加到请求头中
	if crumbField != "" {
		req.Header.Set(crumbField, crumb)
	} else {
		req.Header.Set("Jenkins-Crumb", crumb)
	}
	fmt.Printf("[Jenkins] crumb已添加到请求头, field: %s\n", crumbField)

	fmt.Printf("[Jenkins] 请求URL: %s\n", reqURL)
	fmt.Printf("[Jenkins] 请求头: %+v\n", req.Header)

	resp, err := c.Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 403 {
		return 0, fmt.Errorf("403 Forbidden - crumb验证失败: %s", string(body))
	}

	if resp.StatusCode != 201 {
		return 0, fmt.Errorf("build job failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	// 从响应头获取队列ID
	location := resp.Header.Get("Location")
	if location == "" {
		return 0, fmt.Errorf("no location header in response")
	}

	// 提取队列ID
	queuePath := strings.TrimPrefix(location, c.BaseURL+"/queue/item/")
	queuePath = strings.TrimSuffix(queuePath, "/")
	queueID, err := parseQueueID(queuePath)
	if err != nil {
		return 0, fmt.Errorf("could not parse queue ID from location: %s, error: %v", location, err)
	}

	fmt.Printf("[Jenkins] 构建成功触发，Queue ID: %d\n", queueID)
	return queueID, nil
}

// parseQueueID 从路径解析队列ID
func parseQueueID(path string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(path, "%d", &id)
	return id, err
}

// GetBuildInfo 获取构建信息
func (c *Client) GetBuildInfo(jobPath string, buildNum int) (map[string]interface{}, error) {
	jobPath = strings.TrimPrefix(jobPath, "/")
	reqURL := fmt.Sprintf("%s/job/%s/%d/api/json", c.BaseURL, strings.ReplaceAll(jobPath, "/", "/job/"), buildNum)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get build info failed with status: %d", resp.StatusCode)
	}

	var buildInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&buildInfo); err != nil {
		return nil, err
	}

	return buildInfo, nil
}

// GetBuildStatus 获取构建状态
func (c *Client) GetBuildStatus(jobPath string, buildNum int) (*BuildStatus, error) {
	jobPath = strings.TrimPrefix(jobPath, "/")
	reqURL := fmt.Sprintf("%s/job/%s/%d/api/json", c.BaseURL, strings.ReplaceAll(jobPath, "/", "/job/"), buildNum)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get build status failed with status: %d", resp.StatusCode)
	}

	var buildInfo struct {
		Result            string `json:"result"`
		Building          bool   `json:"building"`
		EstimatedDuration int64  `json:"estimatedDuration"`
		Timestamp         int64  `json:"timestamp"`
		URL               string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&buildInfo); err != nil {
		return nil, err
	}

	status := &BuildStatus{
		Phase:             "UNKNOWN",
		Result:            buildInfo.Result,
		URL:               buildInfo.URL,
		Duration:          buildInfo.EstimatedDuration,
		EstimatedDuration: buildInfo.EstimatedDuration,
		BuildTimestamp:    buildInfo.Timestamp,
	}

	if buildInfo.Building {
		status.Phase = "BUILDING"
	} else if buildInfo.Result != "" {
		status.Phase = "COMPLETED"
	} else {
		status.Phase = "QUEUED"
	}

	return status, nil
}

// GetQueueItemInfo 获取队列项信息
func (c *Client) GetQueueItemInfo(queueID int64) (*QueueItem, error) {
	reqURL := fmt.Sprintf("%s/queue/item/%d/api/json", c.BaseURL, queueID)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get queue item info failed with status: %d", resp.StatusCode)
	}

	var queueItem QueueItem
	if err := json.NewDecoder(resp.Body).Decode(&queueItem); err != nil {
		return nil, err
	}

	return &queueItem, nil
}

// GetJobInfo 获取Job信息
func (c *Client) GetJobInfo(jobName string) (*JobInfo, error) {
	jobName = strings.TrimPrefix(jobName, "/")
	reqURL := fmt.Sprintf("%s/job/%s/api/json", c.BaseURL, jobName)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get job info failed with status: %d", resp.StatusCode)
	}

	var jobInfo JobInfo
	if err := json.NewDecoder(resp.Body).Decode(&jobInfo); err != nil {
		return nil, err
	}

	return &jobInfo, nil
}

// JobInfo Job信息
type JobInfo struct {
	Name      string `json:"name"`
	Buildable bool   `json:"buildable"`
	LastBuild *struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	} `json:"lastBuild"`
	Builds []struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	} `json:"builds"`
}

// GetBuildArtifacts 获取构建产物
func (c *Client) GetBuildArtifacts(jobName string, buildNum int) ([]Artifact, error) {
	reqURL := fmt.Sprintf("%s/job/%s/%d/artifact/api/json", c.BaseURL, jobName, buildNum)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get build artifacts failed with status: %d", resp.StatusCode)
	}

	var artifactsResp struct {
		Artifacts []Artifact `json:"artifacts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&artifactsResp); err != nil {
		return nil, err
	}

	return artifactsResp.Artifacts, nil
}

// Artifact 产物信息
type Artifact struct {
	Name string `json:"fileName"`
	Path string `json:"relativePath"`
	Size int64  `json:"size"`
}

// GetConsoleLog 获取控制台日志
func (c *Client) GetConsoleLog(jobName string, buildNum int) (string, error) {
	reqURL := fmt.Sprintf("%s/job/%s/%d/consoleText", c.BaseURL, jobName, buildNum)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("get console log failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// TriggerBuild 触发构建（旧版本兼容）
func (c *Client) TriggerBuild(jobName string, params BuildParams) (int64, *BuildStatus, error) {
	buildParams := map[string]string{
		"app":   params.APP,   // Jenkins 参数名为小写
		"tag":   params.TAG,   // Jenkins 参数名为小写
		"scope": params.SCOPE, // Jenkins 参数名为小写
	}

	queueID, err := c.BuildJobWithParams(jobName, buildParams)
	if err != nil {
		return 0, nil, err
	}

	// 返回初始状态
	status := &BuildStatus{
		Phase: "QUEUED",
		URL:   fmt.Sprintf("%s/queue/item/%d", c.BaseURL, queueID),
	}

	return queueID, status, nil
}

// ParseBuildTimestampFromLog 从构建日志解析时间戳
func ParseBuildTimestampFromLog(log string) []string {
	var timestamps []string
	// 尝试匹配常见的时间戳格式
	// 格式1: [YYYY-MM-DD HH:MM:SS]
	re := regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]`)
	matches := re.FindAllStringSubmatch(log, -1)
	for _, match := range matches {
		if len(match) > 1 {
			timestamps = append(timestamps, match[1])
		}
	}
	return timestamps
}

// ParseArtifactNameFromLog 从构建日志解析产物名称
func ParseArtifactNameFromLog(log string) []string {
	var artifacts []string
	// 尝试匹配产物名称
	re := regexp.MustCompile(`Archiving artifact:\s*(.+?)(?:\n|$)`)
	matches := re.FindAllStringSubmatch(log, -1)
	for _, match := range matches {
		if len(match) > 1 {
			artifacts = append(artifacts, strings.TrimSpace(match[1]))
		}
	}
	// 尝试匹配其他格式
	re2 := regexp.MustCompile(`artifact:\s*(\S+\.(zip|tar|gz|jar|war|exe))`)
	matches2 := re2.FindAllStringSubmatch(log, -1)
	for _, match := range matches2 {
		if len(match) > 1 {
			artifacts = append(artifacts, strings.TrimSpace(match[1]))
		}
	}
	return artifacts
}

// GetViewJobs 获取视图中的所有Job
func (c *Client) GetViewJobs(viewName string) (*ViewInfo, error) {
	reqURL := fmt.Sprintf("%s/view/%s/api/json?tree=jobs[name]", c.BaseURL, viewName)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("get view jobs failed with status: %d", resp.StatusCode)
	}
	var result ViewInfo
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ViewInfo 视图信息
type ViewInfo struct {
	Name string `json:"name"`
	Jobs []struct {
		Name  string `json:"name"`
		URL   string `json:"url"`
		Color string `json:"color"`
	} `json:"jobs"`
}

// CreateView 创建Jenkins视图
func (c *Client) CreateView(viewName, configXML string) error {
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		return err
	}
	reqURL := fmt.Sprintf("%s/createView?name=%s", c.BaseURL, viewName)
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(configXML))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/xml")

	// 添加对应的 session cookies
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		fmt.Printf("[Jenkins] CreateView 已添加 %d 个 session cookies\n", len(cookies))
	}

	if crumb != "" {
		req.Header.Set("Jenkins-Crumb", crumb)
		// 对于XML请求，也需要在请求头中设置crumb字段（如果知道字段名）
		if crumbField != "" && crumbField != "Jenkins-Crumb" {
			req.Header.Set(crumbField, crumb)
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return fmt.Errorf("create view failed with status: %d", resp.StatusCode)
	}
	return nil
}

// DeleteJob 删除Job
func (c *Client) DeleteJob(jobName string) error {
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		return err
	}
	reqURL := fmt.Sprintf("%s/job/%s/doDelete", c.BaseURL, jobName)
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)

	// 添加对应的 session cookies
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		fmt.Printf("[Jenkins] DeleteJob 已添加 %d 个 session cookies\n", len(cookies))
	}

	if crumb != "" {
		req.Header.Set("Jenkins-Crumb", crumb)
		// 对于空体请求，也要设置正确的crumb字段（如果知道字段名）
		if crumbField != "" && crumbField != "Jenkins-Crumb" {
			req.Header.Set(crumbField, crumb)
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return fmt.Errorf("delete job failed with status: %d", resp.StatusCode)
	}
	return nil
}

// DeleteView 删除视图
func (c *Client) DeleteView(viewName string) error {
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		return err
	}
	reqURL := fmt.Sprintf("%s/view/%s/doDelete", c.BaseURL, viewName)
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)

	// 添加对应的 session cookies
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		fmt.Printf("[Jenkins] DeleteView 已添加 %d 个 session cookies\n", len(cookies))
	}

	if crumb != "" {
		req.Header.Set("Jenkins-Crumb", crumb)
		// 对于空体请求，也要设置正确的crumb字段（如果知道字段名）
		if crumbField != "" && crumbField != "Jenkins-Crumb" {
			req.Header.Set(crumbField, crumb)
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return fmt.Errorf("delete view failed with status: %d", resp.StatusCode)
	}
	return nil
}

// SSHCredentialRequest SSH凭据请求
// SSHCredentialRequest SSH凭据请求
// 支持多种字段名以兼容不同的前端
type SSHCredentialRequest struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	PrivateKey  string `json:"privateKey"`
	PrivateKey2 string `json:"privateKeySource"` // 兼容旧版前端
	PrivateKey3 string `json:"private_key"`      // 兼容前端 snake_case
	Passphrase  string `json:"passphrase,omitempty"`
	Description string `json:"description"`
}

// GetPrivateKey 获取私钥，支持多种字段名
func (r *SSHCredentialRequest) GetPrivateKey() string {
	if r.PrivateKey != "" {
		return r.PrivateKey
	}
	if r.PrivateKey2 != "" {
		return r.PrivateKey2
	}
	return r.PrivateKey3
}

// CreateSSHCredential 创建SSH凭据
func (c *Client) CreateSSHCredential(cred SSHCredentialRequest) error {
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		return err
	}

	// 使用 Jenkins 预期的表单格式
	formData := url.Values{}
	formData.Set("json", fmt.Sprintf(`{
		"credentials": {
			"stapler-class": "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey",
			"$class": "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey",
			"scope": "GLOBAL",
			"id": "%s",
			"username": "%s",
			"usernameSecret": false,
			"privateKeySource": {
				"stapler-class": "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey$DirectEntryPrivateKeySource",
				"$class": "com.cloudbees.jenkins.plugins.sshcredentials.impl.BasicSSHUserPrivateKey$DirectEntryPrivateKeySource",
				"privateKey": "%s"
			},
			"passphrase": "%s",
			"description": "%s"
		}
	}`, cred.ID, cred.Username, strings.ReplaceAll(cred.GetPrivateKey(), "\n", "\\n"), cred.Passphrase, cred.Description))

	reqURL := fmt.Sprintf("%s/credentials/store/system/domain/_/createCredentials", c.BaseURL)
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// 添加 CSRF 保护
	if crumb != "" {
		req.Header.Set("Jenkins-Crumb", crumb)
		if crumbField != "" && crumbField != "Jenkins-Crumb" {
			req.Header.Set(crumbField, crumb)
		}
	}

	// 添加 session cookies
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		fmt.Printf("[Jenkins] CreateSSHCredential 已添加 %d 个 session cookies\n", len(cookies))
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != 200 && resp.StatusCode != 302 && resp.StatusCode != 201 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create ssh credential failed with status: %d, body: %s", resp.StatusCode, string(body))
	}
	return nil
}

// GetJobConfigXML 获取Job配置XML
func (c *Client) GetJobConfigXML(jobName string) (string, error) {
	reqURL := fmt.Sprintf("%s/job/%s/config.xml", c.BaseURL, jobName)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return "", err
	}

	// 使用API Token进行认证 (SetBasicAuth is sufficient)
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("get job config failed with status: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// JobExists 检查Job是否存在
func (c *Client) JobExists(jobName string) bool {
	reqURL := fmt.Sprintf("%s/job/%s/api/json", c.BaseURL, jobName)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return false
	}
	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == 200
}

// CreateJob 创建Job
func (c *Client) CreateJob(jobName, configXML string) error {
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		return err
	}
	reqURL := fmt.Sprintf("%s/createItem?name=%s", c.BaseURL, jobName)
	req, err := http.NewRequest("POST", reqURL, strings.NewReader(configXML))
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/xml")

	// 添加对应的 session cookies
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		fmt.Printf("[Jenkins] CreateJob 已添加 %d 个 session cookies\n", len(cookies))
	}

	if crumb != "" {
		req.Header.Set("Jenkins-Crumb", crumb)
		// 对于XML请求，也要设置正确的crumb字段（如果知道字段名）
		if crumbField != "" && crumbField != "Jenkins-Crumb" {
			req.Header.Set(crumbField, crumb)
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		fmt.Printf("[Jenkins] CreateJob 请求失败: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	fmt.Printf("[Jenkins] CreateJob 响应状态: %d, jobName: %s\n", resp.StatusCode, jobName)

	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Jenkins] CreateJob 失败响应体: %s\n", string(body))
		return fmt.Errorf("create job failed with status: %d", resp.StatusCode)
	}
	fmt.Printf("[Jenkins] CreateJob 成功: %s\n", jobName)
	return nil
}

// AddJobToView 将Job添加到视图
func (c *Client) AddJobToView(viewName, jobName string) error {
	// 获取 crumb 和对应的 cookies（每个请求独立获取，避免竞态条件）
	fmt.Printf("[Jenkins] AddJobToView 开始: viewName=%s, jobName=%s\n", viewName, jobName)
	crumb, crumbField, cookies, err := c.GetCrumbWithCookies()
	if err != nil {
		fmt.Printf("[Jenkins] AddJobToView 获取crumb失败: %v\n", err)
		return err
	}
	fmt.Printf("[Jenkins] AddJobToView 获取crumb成功: crumb=%s, field=%s, cookies=%d\n", crumb[:16]+"...", crumbField, len(cookies))

	reqURL := fmt.Sprintf("%s/view/%s/addJobToView?name=%s", c.BaseURL, viewName, jobName)
	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(c.Username, c.Password)

	// 添加对应的 session cookies（与 crumb 来自同一个请求）
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		fmt.Printf("[Jenkins] AddJobToView 已添加 %d 个 session cookies\n", len(cookies))
	} else {
		fmt.Printf("[Jenkins] AddJobToView 警告: 没有 session cookies\n")
	}

	if crumb != "" {
		req.Header.Set("Jenkins-Crumb", crumb)
		// 对于空体请求，也要设置正确的crumb字段（如果知道字段名）
		if crumbField != "" && crumbField != "Jenkins-Crumb" {
			req.Header.Set(crumbField, crumb)
		}
	}
	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 打印响应状态
	fmt.Printf("[Jenkins] AddJobToView 响应状态: %d\n", resp.StatusCode)

	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		// 读取响应体以获取更多信息
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("[Jenkins] AddJobToView 失败响应体: %s\n", string(body))
		return fmt.Errorf("add job to view failed with status: %d", resp.StatusCode)
	}
	return nil
}

// ApproveAllPendingScripts 保留旧接口，返回已批准的总数量。
func (c *Client) ApproveAllPendingScripts() (int, error) {
	result, err := c.ApprovePendingScriptApprovalItems()
	return result.ApprovedCount(), err
}

// ApprovePendingScriptApprovalItems 批准 Jenkins Script Approval 页面中的待处理脚本和签名。
func (c *Client) ApprovePendingScriptApprovalItems() (ScriptApprovalResult, error) {
	page, err := c.fetchScriptApprovalPage()
	if err != nil {
		return ScriptApprovalResult{}, err
	}

	result := ScriptApprovalResult{}
	for _, hash := range page.PendingScripts {
		if err := c.invokeScriptApprovalMethod(page.ProxyURL, page.ProxyCrumb, "approveScript", []string{hash}, page.Cookies); err != nil {
			return result, fmt.Errorf("approve pending script %s failed: %w", hash, err)
		}
		result.ApprovedScripts++
	}
	for _, signature := range page.PendingSignatures {
		if err := c.invokeScriptApprovalMethod(page.ProxyURL, page.ProxyCrumb, "approveSignature", []string{signature}, page.Cookies); err != nil {
			return result, fmt.Errorf("approve pending signature %s failed: %w", signature, err)
		}
		result.ApprovedSignatures++
	}
	for _, signature := range page.PendingACLSignatures {
		if err := c.invokeScriptApprovalMethod(page.ProxyURL, page.ProxyCrumb, "aclApproveSignature", []string{signature}, page.Cookies); err != nil {
			return result, fmt.Errorf("approve pending ACL signature %s failed: %w", signature, err)
		}
		result.ApprovedSignatures++
	}

	return result, nil
}

type scriptApprovalPage struct {
	ProxyURL             string
	ProxyCrumb           string
	PendingScripts       []string
	PendingSignatures    []string
	PendingACLSignatures []string
	Cookies              []*http.Cookie
}

func (c *Client) fetchScriptApprovalPage() (*scriptApprovalPage, error) {
	reqURL := fmt.Sprintf("%s/scriptApproval/", c.BaseURL)
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(c.Username, c.Password)
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get script approval page failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read script approval page failed: %v", err)
	}

	html := string(body)
	proxyRe := regexp.MustCompile(`makeStaplerProxy\('([^']+)'\s*,\s*'([^']*)'`)
	proxyMatch := proxyRe.FindStringSubmatch(html)
	if len(proxyMatch) < 3 {
		return nil, fmt.Errorf("parse script approval proxy failed")
	}

	return &scriptApprovalPage{
		ProxyURL:             proxyMatch[1],
		ProxyCrumb:           proxyMatch[2],
		PendingScripts:       extractSingleQuotedArgs(html, regexp.MustCompile(`approveScript\('((?:\\'|[^'])+)'\)`)),
		PendingSignatures:    extractSingleQuotedArgs(html, regexp.MustCompile(`approveSignature\('((?:\\'|[^'])+)'\s*,`)),
		PendingACLSignatures: extractSingleQuotedArgs(html, regexp.MustCompile(`aclApproveSignature\('((?:\\'|[^'])+)'\s*,`)),
		Cookies:              resp.Cookies(),
	}, nil
}

func extractSingleQuotedArgs(input string, re *regexp.Regexp) []string {
	matches := re.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		value := strings.ReplaceAll(match[1], `\'`, `'`)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		values = append(values, value)
	}
	return values
}

func (c *Client) invokeScriptApprovalMethod(proxyURL, proxyCrumb, method string, args []string, cookies []*http.Cookie) error {
	body, err := json.Marshal(args)
	if err != nil {
		return err
	}

	reqURL := c.BaseURL + "/" + strings.Trim(strings.TrimLeft(proxyURL, "/"), "/") + "/" + method
	req, err := http.NewRequest("POST", reqURL, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.SetBasicAuth(c.Username, c.Password)
	req.Header.Set("Content-Type", "application/x-stapler-method-invocation;charset=UTF-8")
	if proxyCrumb != "" {
		req.Header.Set("Crumb", proxyCrumb)
		req.Header.Set("Jenkins-Crumb", proxyCrumb)
	}
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusFound {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("invoke %s failed with status: %d, body: %s", method, resp.StatusCode, string(body))
	}

	return nil
}
