package main

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/ar_log"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/ar_trace"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/field"
	"go.opentelemetry.io/otel/trace"
	"time"
)

func main() {
	ctx := context.Background()
	// 第一个参数为微服务名称、第二个参数为微服务版本、第三个参数为微服务实例标识
	arLogger := ar_log.InitARLogger("my-service", "7.0.5.1", "instance1")
	// 第一个参数为微服务名称、第二个参数为微服务版本、第三个参数为微服务实例标识
	arTracerProvider := ar_trace.InitARTracer("my-service", "7.0.5.1", "instance1")
	defer ar_trace.StopARTracer(arTracerProvider, ctx)

	// 业务代码
	ctx, _ = add(ctx, 1, 2)

	// arLogger 记录日志，同时关联链路数据。
	arLogger.Info("This is an info message", field.WithContext(ctx))

	// 日志记录有延迟，等待一下再退出程序
	time.Sleep(5 * time.Second)
}

func add(ctx context.Context, x, y int64) (context.Context, int64) {
	// 第二个参数为span名称、第三个参数为span类型：1表示服务内部调用
	ctx, span := ar_trace.Tracer.Start(ctx, "加法", trace.WithSpanKind(1))
	defer span.End()
	// 第一个参数为span状态：2表示成功
	span.SetStatus(2, "成功计算加法")
	return ctx, x + y
}
