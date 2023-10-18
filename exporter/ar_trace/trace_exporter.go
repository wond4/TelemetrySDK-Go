package ar_trace

import (
	"bytes"
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/common"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/public"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/resource"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/version"
	"encoding/json"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"os"
	"time"
)

// 跨包实现接口占位用。
var _ sdktrace.SpanExporter = (*TraceExporter)(nil)

// Tracer 是一个全局变量，用于在业务代码中生产Span。
var Tracer = otel.GetTracerProvider().Tracer(
	version.TraceInstrumentationName,
	trace.WithInstrumentationVersion(version.TelemetrySDKVersion),
	trace.WithSchemaURL(version.TraceInstrumentationURL),
)

// TraceExporter 导出数据到AnyRobot Feed Ingester的 Event 数据接收器。
type TraceExporter struct {
	*public.Exporter
}

// ExportSpans 批量发送AnyRobotSpans到AnyRobot Feed Ingester的Trace数据接收器。
func (e *TraceExporter) ExportSpans(ctx context.Context, traces []sdktrace.ReadOnlySpan) error {
	if len(traces) == 0 {
		return nil
	}
	arTrace := common.AnyRobotTraceFromReadOnlyTrace(traces)
	file := bytes.NewBuffer([]byte{})
	encoder := json.NewEncoder(file)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "\t")
	if err := encoder.Encode(arTrace); err != nil {
		return err
	}
	return e.ExportData(ctx, file.Bytes())
}

// NewExporter 创建已启动的 TraceExporter 。
func NewExporter(c public.Client) *TraceExporter {
	return &TraceExporter{
		public.NewExporter(c),
	}
}

// TraceResource 传入 Trace 的默认Resource。
func TraceResource() *sdkresource.Resource {
	return resource.TraceResource()
}

// InitARTracer 初始化上报到AnyRobot的链路数据记录器
// ServerName 微服务名称
// ServerVersion 微服务版本
// ServerInstance 微服务实例标识
func InitARTracer(ServerName string, ServerVersion string, ServerInstance string) *sdktrace.TracerProvider {
	traceEnabled := os.Getenv("TELEMETRY_TRACE_ENABLED")
	traceUrl := os.Getenv("TELEMETRY_TRACE_ENDPOINT")

	if traceEnabled == "true" {
		resource.SetServiceName(ServerName)
		resource.SetServiceVersion(ServerVersion)
		resource.SetServiceInstance(ServerInstance)

		traceClient := public.NewHTTPClient(public.WithAnyRobotURL(traceUrl),
			public.WithCompression(1), public.WithTimeout(10*time.Second),
			public.WithRetry(true, 5*time.Second, 30*time.Second, 1*time.Minute))
		traceExporter := NewExporter(traceClient)
		tracerProvider := sdktrace.NewTracerProvider(
			sdktrace.WithBatcher(traceExporter,
				sdktrace.WithMaxExportBatchSize(1000)),
			sdktrace.WithResource(TraceResource()))

		otel.SetTracerProvider(tracerProvider)
		otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

		return tracerProvider
	} else {
		return nil
	}
}

// StopARTracer 关闭 ARTracer
func StopARTracer(tracerProvider *sdktrace.TracerProvider, ctx context.Context) {
	if tracerProvider == nil {
		return
	}

	if err := tracerProvider.Shutdown(ctx); err != nil {
		panic(err)
	}
}
