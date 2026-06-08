//go:build ignore
// +build ignore

package main

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"time"

	"github.com/edy/ops-platform/pkg/jenkins"
)

func main() {
	baseURL := os.Getenv("JENKINS_URL")
	username := os.Getenv("JENKINS_USERNAME")
	token := os.Getenv("JENKINS_TOKEN")
	jobName := os.Getenv("JENKINS_JOB")

	if baseURL == "" || username == "" || token == "" || jobName == "" {
		fmt.Fprintln(os.Stderr, "请设置 JENKINS_URL、JENKINS_USERNAME、JENKINS_TOKEN、JENKINS_JOB 后再手动运行该脚本")
		os.Exit(1)
	}

	// 手动验证脚本，不参与默认 go test / go build 流程。
	client := jenkins.NewClient(baseURL, username, token, 30*time.Second)

	// 测试获取 crumb
	fmt.Println("1. 测试获取 Jenkins Crumb...")
	crumb, _, err := client.GetCrumb()
	if err != nil {
		fmt.Printf("   获取 crumb 失败: %v\n", err)
	} else {
		fmt.Printf("   获取 crumb 成功: %s\n", crumb[:20]+"...")
	}

	// 测试构建 - 使用真实的 job 名称
	fmt.Println("2. 测试触发 Jenkins 构建...")
	params := jenkins.BuildParams{
		APP:   "test-app",
		TAG:   "2f_dev",
		SCOPE: "all",
	}
	queueID, buildStatus, err := client.TriggerBuild(jobName, params)
	if err != nil {
		fmt.Printf("   构建触发失败: %v\n", err)
	} else {
		fmt.Printf("   构建触发成功! queueID: %d, buildStatus: %+v\n", queueID, buildStatus)
	}
}

// testManualCookie 测试手动处理 cookie
func testManualCookie() {
	baseURL := os.Getenv("JENKINS_URL")
	username := os.Getenv("JENKINS_USERNAME")
	token := os.Getenv("JENKINS_TOKEN")

	if baseURL == "" || username == "" || token == "" {
		fmt.Fprintln(os.Stderr, "请设置 JENKINS_URL、JENKINS_USERNAME、JENKINS_TOKEN 后再手动运行该脚本")
		return
	}

	// 创建 cookie jar
	jar, _ := cookiejar.New(nil)

	// 创建 http client
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
	}

	// 1. 获取 crumb
	fmt.Println("1. 获取 crumb...")
	crumbURL := baseURL + "/crumbIssuer/api/json"
	req1, _ := http.NewRequest("GET", crumbURL, nil)
	req1.SetBasicAuth(username, token)

	resp1, err := httpClient.Do(req1)
	if err != nil {
		fmt.Printf("   请求失败: %v\n", err)
		return
	}
	defer resp1.Body.Close()

	// 打印 cookies
	fmt.Printf("   响应状态: %d\n", resp1.StatusCode)
	fmt.Printf("   Cookie jar 中的 cookies: %+v\n", jar)

	for _, cookie := range jar.Cookies(&url.URL{Scheme: "http", Host: req1.URL.Host, Path: "/"}) {
		fmt.Printf("   - %s = %s\n", cookie.Name, cookie.Value[:20]+"...")
	}

	// 解析 crumb
	// ... 省略解析代码
}
