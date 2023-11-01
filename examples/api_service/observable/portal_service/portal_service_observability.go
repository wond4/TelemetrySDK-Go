package main

import (
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_trace"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
		maskedName := desensitizeUserName(name)
		c.String(http.StatusOK, maskedName)
	})
	_ = r.Run(":50080")
}

// desensitizeUserName 用户名称脱敏
func desensitizeUserName(name string) string {
	runes := []rune(name)

	for i := 1; i < len(runes); i++ {
		runes[i] = '*'
	}

	return string(runes)
}

// getUser 调用其他服务，根据用户ID获取用户名称
func getUser(id string, c *gin.Context) string {
	// 第二个参数为span名称、第三个参数为span类型
	ctx, span := ar_trace.Tracer.Start(c.Request.Context(), "根据用户ID获取用户名称", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	// 第一个参数为span状态、第二个参数为span状态描述
	span.SetStatus(codes.Ok, "")

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	url := fmt.Sprintf("http://127.0.0.1:50081/users/%s", id)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
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
