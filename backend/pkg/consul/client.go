package consul

import (
	"fmt"
	"github.com/hashicorp/consul/api"
)

// Client Consul客户端结构体
type Client struct {
	client *api.Client
}

// NewClient 创建Consul客户端
func NewClient(addr string) (*Client, error) {
	config := &api.Config{
		Address: addr,
	}
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &Client{
		client: client,
	}, nil
}

// GetKVValue 获取指定路径的键值
func (c *Client) GetKVValue(path string) (string, error) {
	pair, _, err := c.client.KV().Get(path, nil)
	if err != nil {
		return "", err
	}
	if pair == nil {
		return "", fmt.Errorf("key %s not found", path)
	}
	return string(pair.Value), nil
}

// GetTagsFromPath 从指定路径获取tags
func (c *Client) GetTagsFromPath(basePath string, appNames []string) (map[string]string, error) {
	tags := make(map[string]string)

	for _, appName := range appNames {
		key := fmt.Sprintf("%s%s", basePath, appName)
		value, err := c.GetKVValue(key)
		if err != nil {
			// 不返回错误，只记录日志并使用默认值
			fmt.Printf("Warning: Could not get tag for %s from path %s: %v", appName, key, err)
			tags[appName] = "latest" // 默认值
		} else {
			tags[appName] = value
		}
	}

	return tags, nil
}

// PutKVValue 设置键值对
func (c *Client) PutKVValue(key, value string) error {
	p := &api.KVPair{Key: key, Value: []byte(value)}
	_, err := c.client.KV().Put(p, nil)
	return err
}

// DeleteKVValue 删除键值对
func (c *Client) DeleteKVValue(key string) error {
	_, err := c.client.KV().Delete(key, nil)
	return err
}