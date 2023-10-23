package main

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/ar_trace"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	ctx := context.Background()
	arTracerProvider := ar_trace.InitARTracer()
	defer ar_trace.StopARTracer(arTracerProvider, ctx)

	// 服务内部函数调用
	ctx, _ = add(ctx, 1, 2)

}

func add(ctx context.Context, x, y int64) (context.Context, int64) {
	// 第二个参数为span名称、第三个参数为span类型
	ctx, span := ar_trace.Tracer.Start(ctx, "加法", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	// 第一个参数为span状态、第二个参数为span状态描述
	span.SetStatus(codes.Ok, "成功计算加法")
	return ctx, x + y
}
