package main

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_trace"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"io"
	"net/http"
)

func main() {
	arTracerProvider := ar_trace.InitARTracer()
	defer ar_trace.StopARTracer(arTracerProvider)
	r := gin.Default()
	r.Use(otelgin.Middleware("my-server-portal"))
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		name := getUser(id, c)
		maskedName := desensitizeUserName(name, c)
		c.String(http.StatusOK, maskedName)
	})
	_ = r.Run(":50080")
}

// desensitizeUserName 用户名称脱敏
func desensitizeUserName(name string, ctx context.Context) string {
	var err error
	newCtx, span := ar_trace.StartInternalSpan(ctx)
	// 结束span时，err如果不为空的话，则span状态设置为error
	defer func() { ar_trace.EndSpan(newCtx, err) }()
	// 不设置span name的话，span name 默认为函数名称
	span.SetName("用户名称脱敏")

	// 将程序错误赋值给err，便于结束span时根据err是否为空设置span状态
	if len(name) == 0 {
		err = errors.New("用户名称为空字符串")
	}

	runes := []rune(name)

	for i := 1; i < len(runes); i++ {
		runes[i] = '*'
	}

	return string(runes)
}

// getUser 调用其他服务，根据用户ID获取用户名称
func getUser(id string, ctx context.Context) string {
	var err error
	newCtx, span := ar_trace.StartInternalSpan(ctx)
	// 结束span时，err如果不为空的话，则span状态设置为error
	defer func() { ar_trace.EndSpan(newCtx, err) }()
	// 不设置span name的话，span name 默认为函数名称
	span.SetName("根据用户ID获取用户名称")

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	url := fmt.Sprintf("http://127.0.0.1:50081/users/%s", id)
	req, _ := http.NewRequestWithContext(newCtx, "GET", url, nil)
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("请求失败:", err)
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("读取响应失败:", err)
		return ""
	}

	return string(body)
}
