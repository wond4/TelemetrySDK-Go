package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	// 创建一个 HTTP 客户端
	client := http.Client{}
	client.Timeout = 5 * time.Second

	// 发起 GET 请求
	resp, err := client.Get("http://127.0.0.1:50080/address")
	if err != nil {
		fmt.Println("请求失败:", err)
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("关闭响应失败:", err)
		}
	}(resp.Body)

	// 读取响应内容
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return
	}

	// 打印响应内容
	fmt.Println(string(body))
}
