package consul

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Service Consul 服务
type Service struct {
	db     *gorm.DB
	client *http.Client
}

// NewService 创建 Consul 服务
func NewService(db *gorm.DB) *Service {
	return &Service{
		db: db,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

// GetConfigs 获取所有 Consul 配置
func (s *Service) GetConfigs() ([]ConsulConfig, error) {
	var configs []ConsulConfig
	err := s.db.Order("is_default DESC, name ASC").Find(&configs).Error
	return configs, err
}

// GetConfig 获取单个配置
func (s *Service) GetConfig(id uint) (*ConsulConfig, error) {
	var config ConsulConfig
	err := s.db.First(&config, id).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// GetDefaultConfig 获取默认配置
func (s *Service) GetDefaultConfig() (*ConsulConfig, error) {
	var config ConsulConfig
	result := s.db.Where("is_default = ?", true).Limit(1).Find(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected > 0 {
		return &config, nil
	}

	result = s.db.Order("id ASC").Limit(1).Find(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, gorm.ErrRecordNotFound
	}

	return &config, nil
}

// CreateConfig 创建配置
func (s *Service) CreateConfig(config *ConsulConfig) error {
	// 如果设置为默认，取消其他默认配置
	if config.IsDefault {
		s.db.Model(&ConsulConfig{}).Where("is_default = ?", true).Update("is_default", false)
	}
	return s.db.Create(config).Error
}

// UpdateConfig 更新配置
func (s *Service) UpdateConfig(config *ConsulConfig) error {
	// 如果设置为默认，取消其他默认配置
	if config.IsDefault {
		s.db.Model(&ConsulConfig{}).Where("id != ? AND is_default = ?", config.ID, true).Update("is_default", false)
	}
	return s.db.Save(config).Error
}

// DeleteConfig 删除配置
func (s *Service) DeleteConfig(id uint) error {
	return s.db.Delete(&ConsulConfig{}, id).Error
}

// TestConnection 测试连接
func (s *Service) TestConnection(config *ConsulConfig) error {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/v1/agent/self", config.Address), nil)
	if err != nil {
		return err
	}

	s.setAuth(req, config)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("连接失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("连接失败: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ListKeys 列出 KV 键
func (s *Service) ListKeys(configID uint, prefix string, recurse bool) ([]string, error) {
	config, err := s.GetConfig(configID)
	if err != nil {
		return nil, err
	}

	// 构建 URL
	apiURL := fmt.Sprintf("%s/v1/kv/%s?dc=%s&keys", config.Address, prefix, config.Datacenter)
	if recurse {
		apiURL += "&recurse"
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	s.setAuth(req, config)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取键列表失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return []string{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取键列表失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var keys []string
	if err := json.Unmarshal(body, &keys); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	return keys, nil
}

func (s *Service) ListProjects(configID uint) ([]string, error) {
	allKeys, err := s.ListKeys(configID, "plugin/", false)
	if err != nil {
		return nil, fmt.Errorf("获取项目列表失败: %v", err)
	}
	return collectProjectNames(allKeys), nil
}

// GetKeyValue 获取键值
func (s *Service) GetKeyValue(configID uint, key string) (*KVItem, error) {
	config, err := s.GetConfig(configID)
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s/v1/kv/%s?dc=%s", config.Address, key, config.Datacenter)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	s.setAuth(req, config)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("获取键值失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("键不存在")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("获取键值失败: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var items []struct {
		LockIndex   uint64 `json:"LockIndex"`
		Key         string `json:"Key"`
		Flags       uint64 `json:"Flags"`
		Value       string `json:"Value"`
		CreateIndex uint64 `json:"CreateIndex"`
		ModifyIndex uint64 `json:"ModifyIndex"`
	}

	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v", err)
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("键不存在")
	}

	item := items[0]

	// Base64 解码
	decoded, err := base64.StdEncoding.DecodeString(item.Value)
	if err != nil {
		return nil, fmt.Errorf("解码值失败: %v", err)
	}

	return &KVItem{
		Key:         item.Key,
		Value:       string(decoded),
		Flags:       item.Flags,
		CreateIndex: item.CreateIndex,
		ModifyIndex: item.ModifyIndex,
	}, nil
}

// PutKeyValue 设置键值
func (s *Service) PutKeyValue(configID uint, key string, value string) error {
	config, err := s.GetConfig(configID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/v1/kv/%s?dc=%s", config.Address, key, config.Datacenter)

	req, err := http.NewRequest("PUT", apiURL, bytes.NewBufferString(value))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	s.setAuth(req, config)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("设置键值失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("设置键值失败: HTTP %d", resp.StatusCode)
	}

	return nil
}

// DeleteKey 删除键
func (s *Service) DeleteKey(configID uint, key string) error {
	config, err := s.GetConfig(configID)
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/v1/kv/%s?dc=%s", config.Address, key, config.Datacenter)

	req, err := http.NewRequest("DELETE", apiURL, nil)
	if err != nil {
		return err
	}

	s.setAuth(req, config)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("删除键失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("删除键失败: HTTP %d", resp.StatusCode)
	}

	return nil
}

// replaceOptions 统一的替换参数
type replaceOptions struct {
	ReplaceRules                []RuleItem
	TagReplacements             []ReplacePair
	ServerReplacements          []ReplacePair
	BranchReplacements          []ReplacePair
	SubmoduleBranchReplacements []ReplacePair
}

// applyReplacements 对值应用所有替换规则（统一替换逻辑，消除重复代码）
func applyReplacements(value string, opts *replaceOptions, errors *[]string) string {
	// 1. 应用高级替换规则
	for _, rule := range opts.ReplaceRules {
		if rule.Type == "regex" {
			re, err := regexp.Compile(rule.OldValue)
			if err != nil {
				if errors != nil {
					*errors = append(*errors, fmt.Sprintf("正则表达式错误: %s", rule.OldValue))
				}
				continue
			}
			value = re.ReplaceAllString(value, rule.NewValue)
		} else {
			value = strings.ReplaceAll(value, rule.OldValue, rule.NewValue)
		}
	}

	// 2. 应用 Tag 模式替换 (精确匹配 key: "value" 形式)
	for _, pair := range opts.TagReplacements {
		if pair.OldPattern != "" && pair.NewPattern != "" {
			// 使用正则表达式匹配完整的 key: "value" 形式，避免部分匹配
			oldEscaped := regexp.QuoteMeta(pair.OldPattern)

			// 分别构建每种格式的正则表达式，确保正确匹配
			// 匹配 tag: "old_value" 格式
			pattern1 := `(?m)\btag\s*:\s*"` + oldEscaped + `"`
			// 匹配 tag='old_value' 格式
			pattern2 := `(?m)\btag\s*=\s*'` + oldEscaped + `'`
			// 匹配 tag="old_value" 格式
			pattern3 := `(?m)\btag\s*=\s*"` + oldEscaped + `"`
			// 匹配 tag: 'old_value' 格式
			pattern4 := `(?m)\btag\s*:\s*'` + oldEscaped + `'`
			// 匹配 tag=old_value 格式（无引号，单词边界）
			pattern5 := `(?m)\btag\s*=\s*` + oldEscaped + `\b`
			// 匹配 tag: old_value 格式（无引号，单词边界）
			pattern6 := `(?m)\btag\s*:\s*` + oldEscaped + `\b`

			// 将所有模式合并为一个正则表达式
			fullPattern := "(" + pattern1 + ")|(" + pattern2 + ")|(" + pattern3 + ")|(" + pattern4 + ")|(" + pattern5 + ")|(" + pattern6 + ")"
			re := regexp.MustCompile(fullPattern)

			// 正确的替换逻辑：直接替换整个匹配部分中的值
			value = re.ReplaceAllStringFunc(value, func(match string) string {
				// 替换匹配字符串中的旧值为新值
				return strings.Replace(match, pair.OldPattern, pair.NewPattern, 1)
			})
		}
	}

	// 3. 应用 Server 模式替换 (精确匹配 key: "value" 形式)
	for _, pair := range opts.ServerReplacements {
		if pair.OldPattern != "" && pair.NewPattern != "" {
			// 使用正则表达式匹配完整的 key: "value" 形式，避免部分匹配
			oldEscaped := regexp.QuoteMeta(pair.OldPattern)

			// 分别构建每种格式的正则表达式，确保正确匹配
			// 匹配 server: "old_value" 格式
			pattern1 := `(?m)\bserver\s*:\s*"` + oldEscaped + `"`
			// 匹配 server='old_value' 格式
			pattern2 := `(?m)\bserver\s*=\s*'` + oldEscaped + `'`
			// 匹配 server="old_value" 格式
			pattern3 := `(?m)\bserver\s*=\s*"` + oldEscaped + `"`
			// 匹配 server: 'old_value' 格式
			pattern4 := `(?m)\bserver\s*:\s*'` + oldEscaped + `'`
			// 匹配 server=old_value 格式（无引号，单词边界）
			pattern5 := `(?m)\bserver\s*=\s*` + oldEscaped + `\b`
			// 匹配 server: old_value 格式（无引号，单词边界）
			pattern6 := `(?m)\bserver\s*:\s*` + oldEscaped + `\b`

			// 将所有模式合并为一个正则表达式
			fullPattern := "(" + pattern1 + ")|(" + pattern2 + ")|(" + pattern3 + ")|(" + pattern4 + ")|(" + pattern5 + ")|(" + pattern6 + ")"
			re := regexp.MustCompile(fullPattern)

			// 正确的替换逻辑：直接替换整个匹配部分中的值
			value = re.ReplaceAllStringFunc(value, func(match string) string {
				// 替换匹配字符串中的旧值为新值
				return strings.Replace(match, pair.OldPattern, pair.NewPattern, 1)
			})
		}
	}

	// 4. 应用 Branch 模式替换 (精确匹配 key: "value" 形式)
	for _, pair := range opts.BranchReplacements {
		if pair.OldPattern != "" && pair.NewPattern != "" {
			// 使用正则表达式匹配完整的 key: "value" 形式，避免部分匹配
			oldEscaped := regexp.QuoteMeta(pair.OldPattern)

			// 分别构建每种格式的正则表达式，确保正确匹配
			// 匹配 branch: "old_value" 格式
			pattern1 := `(?m)\bbranch\s*:\s*"` + oldEscaped + `"`
			// 匹配 branch='old_value' 格式
			pattern2 := `(?m)\bbranch\s*=\s*'` + oldEscaped + `'`
			// 匹配 branch="old_value" 格式
			pattern3 := `(?m)\bbranch\s*=\s*"` + oldEscaped + `"`
			// 匹配 branch: 'old_value' 格式
			pattern4 := `(?m)\bbranch\s*:\s*'` + oldEscaped + `'`
			// 匹配 branch=old_value 格式（无引号，单词边界）
			pattern5 := `(?m)\bbranch\s*=\s*` + oldEscaped + `\b`
			// 匹配 branch: old_value 格式（无引号，单词边界）
			pattern6 := `(?m)\bbranch\s*:\s*` + oldEscaped + `\b`

			// 将所有模式合并为一个正则表达式
			fullPattern := "(" + pattern1 + ")|(" + pattern2 + ")|(" + pattern3 + ")|(" + pattern4 + ")|(" + pattern5 + ")|(" + pattern6 + ")"
			re := regexp.MustCompile(fullPattern)

			// 正确的替换逻辑：直接替换整个匹配部分中的值
			value = re.ReplaceAllStringFunc(value, func(match string) string {
				// 替换匹配字符串中的旧值为新值
				return strings.Replace(match, pair.OldPattern, pair.NewPattern, 1)
			})
		}
	}

	// 5. 应用 SubmoduleBranch 模式替换 (精确匹配 key: "value" 形式)
	for _, pair := range opts.SubmoduleBranchReplacements {
		if pair.OldPattern != "" && pair.NewPattern != "" {
			// 使用正则表达式匹配完整的 key: "value" 形式，避免部分匹配
			oldEscaped := regexp.QuoteMeta(pair.OldPattern)

			// 分别构建每种格式的正则表达式，确保正确匹配
			// 匹配 submoduleBranch: "old_value" 格式
			pattern1 := `(?m)\bsubmoduleBranch\s*:\s*"` + oldEscaped + `"`
			// 匹配 submoduleBranch='old_value' 格式
			pattern2 := `(?m)\bsubmoduleBranch\s*=\s*'` + oldEscaped + `'`
			// 匹配 submoduleBranch="old_value" 格式
			pattern3 := `(?m)\bsubmoduleBranch\s*=\s*"` + oldEscaped + `"`
			// 匹配 submoduleBranch: 'old_value' 格式
			pattern4 := `(?m)\bsubmoduleBranch\s*:\s*'` + oldEscaped + `'`
			// 匹配 submoduleBranch=old_value 格式（无引号，单词边界）
			pattern5 := `(?m)\bsubmoduleBranch\s*=\s*` + oldEscaped + `\b`
			// 匹配 submoduleBranch: old_value 格式（无引号，单词边界）
			pattern6 := `(?m)\bsubmoduleBranch\s*:\s*` + oldEscaped + `\b`

			// 将所有模式合并为一个正则表达式
			fullPattern := "(" + pattern1 + ")|(" + pattern2 + ")|(" + pattern3 + ")|(" + pattern4 + ")|(" + pattern5 + ")|(" + pattern6 + ")"
			re := regexp.MustCompile(fullPattern)

			// 正确的替换逻辑：直接替换整个匹配部分中的值
			value = re.ReplaceAllStringFunc(value, func(match string) string {
				// 替换匹配字符串中的旧值为新值
				return strings.Replace(match, pair.OldPattern, pair.NewPattern, 1)
			})
		}
	}

	return value
}

// CopyKey 复制键（支持替换规则）
func (s *Service) CopyKey(configID uint, req *CopyRequest, operator string) (*CopyResult, error) {
	_, err := s.GetConfig(req.ConfigID)
	if err != nil {
		return nil, err
	}

	result := &CopyResult{
		CopiedKeys: []string{},
		FailedKeys: []string{},
		Errors:     []string{},
	}

	var sourceKeys []string
	if req.Recursive {
		sourceKeys, err = s.ListKeys(configID, req.SourceKey, true)
		if err != nil {
			return nil, fmt.Errorf("获取源键列表失败: %v", err)
		}
	} else {
		sourceKeys = []string{req.SourceKey}
	}

	result.Total = len(sourceKeys)

	op := &CopyOperation{
		ConfigID:  configID,
		SourceKey: req.SourceKey,
		TargetKey: req.TargetKey,
		Status:    "pending",
		Operator:  operator,
	}
	rulesJSON, _ := json.Marshal(req.ReplaceRules)
	op.RulesApplied = string(rulesJSON)
	s.db.Create(op)

	opts := &replaceOptions{
		ReplaceRules:                req.ReplaceRules,
		TagReplacements:             req.TagReplacements,
		ServerReplacements:          req.ServerReplacements,
		BranchReplacements:          req.BranchReplacements,
		SubmoduleBranchReplacements: req.SubmoduleBranchReplacements,
	}

	for _, sourceKey := range sourceKeys {
		kvItem, err := s.GetKeyValue(configID, sourceKey)
		if err != nil {
			result.Failed++
			result.FailedKeys = append(result.FailedKeys, sourceKey)
			result.Errors = append(result.Errors, fmt.Sprintf("获取 %s 失败: %v", sourceKey, err))
			continue
		}

		targetKey := req.TargetKey + strings.TrimPrefix(sourceKey, req.SourceKey)
		newValue := applyReplacements(kvItem.Value, opts, &result.Errors)

		err = s.PutKeyValue(configID, targetKey, newValue)
		if err != nil {
			result.Failed++
			result.FailedKeys = append(result.FailedKeys, sourceKey)
			result.Errors = append(result.Errors, fmt.Sprintf("写入 %s 失败: %v", targetKey, err))
			continue
		}

		result.Success++
		result.CopiedKeys = append(result.CopiedKeys, targetKey)
	}

	if result.Failed > 0 {
		op.Status = "partial"
	} else {
		op.Status = "success"
	}
	op.Message = fmt.Sprintf("成功: %d, 失败: %d", result.Success, result.Failed)
	s.db.Save(op)

	return result, nil
}

// GetOperations 获取操作历史
func (s *Service) GetOperations(configID uint, limit int) ([]CopyOperation, error) {
	var operations []CopyOperation
	query := s.db.Model(&CopyOperation{}).Order("created_at DESC")
	if configID > 0 {
		query = query.Where("config_id = ?", configID)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&operations).Error
	return operations, err
}

// DeleteOperation 删除操作记录
func (s *Service) DeleteOperation(id uint) error {
	return s.db.Delete(&CopyOperation{}, id).Error
}

// GetReplaceRules 获取替换规则列表
func (s *Service) GetReplaceRules() ([]ReplaceRule, error) {
	var rules []ReplaceRule
	err := s.db.Order("sort_order ASC, id ASC").Find(&rules).Error
	return rules, err
}

// CreateReplaceRule 创建替换规则
func (s *Service) CreateReplaceRule(rule *ReplaceRule) error {
	return s.db.Create(rule).Error
}

// UpdateReplaceRule 更新替换规则
func (s *Service) UpdateReplaceRule(rule *ReplaceRule) error {
	return s.db.Save(rule).Error
}

// DeleteReplaceRule 删除替换规则
func (s *Service) DeleteReplaceRule(id uint) error {
	return s.db.Delete(&ReplaceRule{}, id).Error
}

// setAuth 设置认证信息
func (s *Service) setAuth(req *http.Request, config *ConsulConfig) {
	if config.Token != "" {
		req.Header.Set("X-Consul-Token", config.Token)
	}
	if config.Username != "" && config.Password != "" {
		req.SetBasicAuth(config.Username, config.Password)
	}
}

// BuildKVTree 构建 KV 树结构
func (s *Service) BuildKVTree(keys []string, prefix string) []KVNode {
	nodeMap := make(map[string]*KVNode)
	var roots []KVNode

	for _, key := range keys {
		// 移除前缀
		relativePath := strings.TrimPrefix(key, prefix)
		if relativePath == "" {
			continue
		}

		parts := strings.Split(strings.Trim(relativePath, "/"), "/")
		currentPath := prefix

		for i, part := range parts {
			if part == "" {
				continue
			}

			currentPath += part + "/"
			isDir := i < len(parts)-1

			if _, exists := nodeMap[currentPath]; !exists {
				node := KVNode{
					Key:      strings.TrimSuffix(currentPath, "/"),
					Name:     part,
					IsDir:    isDir,
					Children: []KVNode{},
				}
				nodeMap[currentPath] = &node
			}

			// 如果是第一层，添加到 roots
			if i == 0 {
				found := false
				for _, r := range roots {
					if r.Key == nodeMap[currentPath].Key {
						found = true
						break
					}
				}
				if !found {
					roots = append(roots, *nodeMap[currentPath])
				}
			}

			// 添加子节点引用
			if i > 0 {
				parentPath := prefix + strings.Join(parts[:i], "/") + "/"
				if parent, exists := nodeMap[parentPath]; exists {
					found := false
					for _, c := range parent.Children {
						if c.Key == nodeMap[currentPath].Key {
							found = true
							break
						}
					}
					if !found {
						parent.Children = append(parent.Children, *nodeMap[currentPath])
					}
				}
			}
		}
	}

	return roots
}

// EscapeKey 转义键名用于 URL
func EscapeKey(key string) string {
	return url.PathEscape(key)
}

// BatchCopyKeys 批量复制键（参考脚本功能）
func (s *Service) BatchCopyKeys(configID uint, req *BatchCopyRequest, operator string) (*BatchCopyResult, error) {
	result := &BatchCopyResult{
		CopiedKeys: []string{},
		FailedKeys: []string{},
		Errors:     []string{},
	}

	startTime := time.Now()

	var sourceKeys []string
	var err error
	sourceKeys, err = s.ListKeys(configID, req.SourcePrefix, req.Recursive)
	if err != nil {
		return nil, fmt.Errorf("获取源键列表失败：%v", err)
	}

	result.Total = len(sourceKeys)

	opts := &replaceOptions{
		ReplaceRules:                req.ReplaceRules,
		TagReplacements:             req.TagReplacements,
		ServerReplacements:          req.ServerReplacements,
		BranchReplacements:          req.BranchReplacements,
		SubmoduleBranchReplacements: req.SubmoduleBranchReplacements,
	}

	for _, sourceKey := range sourceKeys {
		if sourceKey == "" || sourceKey == req.SourcePrefix {
			continue
		}

		kvItem, err := s.GetKeyValue(configID, sourceKey)
		if err != nil {
			result.Failed++
			result.FailedKeys = append(result.FailedKeys, sourceKey)
			result.Errors = append(result.Errors, fmt.Sprintf("获取 %s 失败：%v", sourceKey, err))
			continue
		}

		targetKey := req.TargetPrefix + strings.TrimPrefix(sourceKey, req.SourcePrefix)
		newValue := applyReplacements(kvItem.Value, opts, &result.Errors)

		err = s.PutKeyValue(configID, targetKey, newValue)
		if err != nil {
			result.Failed++
			result.FailedKeys = append(result.FailedKeys, sourceKey)
			result.Errors = append(result.Errors, fmt.Sprintf("写入 %s 失败：%v", targetKey, err))
			continue
		}

		result.Success++
		result.CopiedKeys = append(result.CopiedKeys, targetKey)
	}

	result.ElapsedTime = time.Since(startTime).String()
	return result, nil
}

// BatchCopyAllProjects 批量复制所有项目（一键复制）
func (s *Service) BatchCopyAllProjects(configID uint, req *BatchCopyAllProjectsRequest, operator string) (*BatchCopyResult, error) {
	result := &BatchCopyResult{
		CopiedKeys: []string{},
		FailedKeys: []string{},
		Errors:     []string{},
	}

	sourceSuffix := strings.TrimSpace(req.SourceSuffix)
	targetSuffix := strings.TrimSpace(req.TargetSuffix)
	replaceInPlace := req.ReplaceInPlace || sourceSuffix == targetSuffix

	if sourceSuffix == "" || targetSuffix == "" {
		return nil, fmt.Errorf("源后缀和目标后缀不能为空")
	}
	if sourceSuffix == targetSuffix && !replaceInPlace {
		return nil, fmt.Errorf("源后缀和目标后缀相同时，必须启用原后缀内替换")
	}

	startTime := time.Now()

	op := &CopyOperation{
		ConfigID:  configID,
		SourceKey: fmt.Sprintf("plugin/*/%s", sourceSuffix),
		TargetKey: fmt.Sprintf("plugin/*/%s", targetSuffix),
		Status:    "pending",
		Operator:  operator,
	}
	if replaceInPlace {
		op.Message = "原后缀内替换"
	}
	rulesJSON, _ := json.Marshal(req.ReplaceRules)
	op.RulesApplied = string(rulesJSON)
	s.db.Create(op)

	allKeys, err := s.ListKeys(configID, "plugin/", false)
	if err != nil {
		op.Status = "failed"
		op.Message = fmt.Sprintf("获取项目列表失败：%v", err)
		s.db.Save(op)
		return nil, fmt.Errorf("获取项目列表失败：%v", err)
	}

	var projects []string
	projectNames := collectProjectNames(allKeys)
	projectSet := make(map[string]struct{}, len(projectNames))
	for _, project := range projectNames {
		projectSet[project] = struct{}{}
	}
	if len(req.Projects) > 0 {
		for _, project := range req.Projects {
			if _, ok := projectSet[project]; ok {
				projects = append(projects, project)
			}
		}
	} else {
		projects = projectNames
	}

	opts := &replaceOptions{
		ReplaceRules:                req.ReplaceRules,
		TagReplacements:             req.TagReplacements,
		ServerReplacements:          req.ServerReplacements,
		BranchReplacements:          req.BranchReplacements,
		SubmoduleBranchReplacements: req.SubmoduleBranchReplacements,
	}

	result.Total = 0
	sourcePrefix := "plugin/%s/" + sourceSuffix
	targetPrefix := "plugin/%s/" + targetSuffix

	for _, project := range projects {
		srcPrefix := fmt.Sprintf(sourcePrefix, project)
		tgtPrefix := fmt.Sprintf(targetPrefix, project)

		sourceKeys, err := s.ListKeys(configID, srcPrefix, req.Recursive)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("获取项目 %s 的键列表失败：%v", project, err))
			continue
		}

		sourceKeys = filterCopySourceKeys(sourceKeys, srcPrefix)
		result.Total += len(sourceKeys)

		for _, sourceKey := range sourceKeys {

			kvItem, err := s.GetKeyValue(configID, sourceKey)
			if err != nil {
				result.Failed++
				result.FailedKeys = append(result.FailedKeys, sourceKey)
				result.Errors = append(result.Errors, fmt.Sprintf("获取 %s 失败：%v", sourceKey, err))
				continue
			}

			targetKey := tgtPrefix + strings.TrimPrefix(sourceKey, srcPrefix)
			newValue := applyReplacements(kvItem.Value, opts, &result.Errors)

			err = s.PutKeyValue(configID, targetKey, newValue)
			if err != nil {
				result.Failed++
				result.FailedKeys = append(result.FailedKeys, sourceKey)
				result.Errors = append(result.Errors, fmt.Sprintf("写入 %s 失败：%v", targetKey, err))
				continue
			}

			result.Success++
			result.CopiedKeys = append(result.CopiedKeys, targetKey)
		}
	}

	if result.Failed > 0 {
		op.Status = "partial"
	} else {
		op.Status = "success"
	}
	op.Message = fmt.Sprintf("成功: %d, 失败: %d", result.Success, result.Failed)
	s.db.Save(op)

	result.ElapsedTime = time.Since(startTime).String()
	return result, nil
}

// ListProjectSuffixKeys 查询所有项目中指定后缀的 Key 列表
func (s *Service) ListProjectSuffixKeys(configID uint, suffix string) ([]string, error) {
	allKeys, err := s.ListKeys(configID, "plugin/", false)
	if err != nil {
		return nil, fmt.Errorf("获取项目列表失败：%v", err)
	}

	var matchedKeys []string
	for _, project := range collectProjectNames(allKeys) {
		prefix := fmt.Sprintf("plugin/%s/%s", project, suffix)
		keys, err := s.ListKeys(configID, prefix, true)
		if err != nil {
			continue
		}
		matchedKeys = append(matchedKeys, keys...)
	}

	return matchedKeys, nil
}

// BatchDeleteKeys 批量删除 KV 键
func (s *Service) BatchDeleteKeys(configID uint, keys []string) (*BatchDeleteResult, error) {
	result := &BatchDeleteResult{
		DeletedKeys: []string{},
		FailedKeys:  []string{},
		Errors:      []string{},
		Total:       len(keys),
	}

	for _, key := range keys {
		err := s.DeleteKey(configID, key)
		if err != nil {
			result.Failed++
			result.FailedKeys = append(result.FailedKeys, key)
			result.Errors = append(result.Errors, fmt.Sprintf("删除 %s 失败：%v", key, err))
		} else {
			result.Deleted++
			result.DeletedKeys = append(result.DeletedKeys, key)
		}
	}

	return result, nil
}
