package ar_log

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/public"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/resource"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/encoder"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/exporter"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/field"
	spanLog "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/log"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/open_standard"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/runtime"
	"os"
	"time"
)

// 跨包实现接口占位用。
var _ exporter.LogExporter = (*SpanExporter)(nil)
var _ exporter.SyncExporter = (*syncExporter)(nil)

// SpanExporter 导出数据到AnyRobot Feed Ingester的 Log 数据接收器。
type SpanExporter struct {
	*public.Exporter
}

// ExportLogs 批量发送 log 到AnyRobot Feed Ingester的 Log 数据接收器。
func (e *SpanExporter) ExportLogs(ctx context.Context, logs []byte) error {
	return e.ExportData(ctx, logs)
}

// NewExporter 创建已启动的 LogExporter。
func NewExporter(c public.Client) *SpanExporter {
	return &SpanExporter{
		public.NewExporter(c),
	}
}

// syncExporter 同步导出数据到AnyRobot Feed Ingester的 Log 数据接收器。
type syncExporter struct {
	*public.SyncExporter
}

// ExportLogs 同步发送 log 到AnyRobot Feed Ingester的 Log 数据接收器。
func (s *syncExporter) ExportLogs(ctx context.Context, logs []byte) error {
	return s.ExportData(ctx, logs)
}

// NewSyncExporter 创建已启动的 LogExporter。
func NewSyncExporter(c public.SyncClient) *syncExporter {
	return &syncExporter{
		public.NewSyncExporter(c),
	}
}

// InitARLogger 初始化上报到AnyRobot的日志记录器
// ServerName 微服务名称
// ServerVersion 微服务版本
// ServerInstance 微服务实例标识
// logLevel 日志等级
func InitARLogger() spanLog.Logger {
	serverName := os.Getenv("TELEMETRY_SERVICE_NAME")
	serverVersion := os.Getenv("TELEMETRY_SERVICE_VERSION")
	serverInstance := os.Getenv("HOSTNAME")
	logEnabled := os.Getenv("TELEMETRY_LOG_ENABLED")
	logLevel := os.Getenv("TELEMETRY_LOG_LEVEL")
	if logEnabled != "true" {
		logLevel = "off"
	}

	// 初始化ar_log
	var ARLogger = spanLog.NewSamplerLogger(spanLog.WithSample(1.0), spanLog.WithLevel(getLogLevel(logLevel)))

	// 设置微服务相关信息
	resource.SetServiceName(serverName)
	resource.SetServiceVersion(serverVersion)
	resource.SetServiceInstance(serverInstance)

	// 设置日志打印标准输出
	systemLogExporter := exporter.GetRealTimeExporter()
	systemLogWriter := open_standard.OpenTelemetryWriter(
		encoder.NewJsonEncoderWithExporters(systemLogExporter),
		resource.LogResource())
	systemLogRunner := runtime.NewRuntime(systemLogWriter, field.NewSpanFromPool)
	systemLogRunner.SetUploadInternalAndMaxLog(3*time.Second, 10)

	go systemLogRunner.Run()
	ARLogger.SetLevel(getLogLevel(logLevel))
	ARLogger.SetRuntime(systemLogRunner)

	return ARLogger
}

// getLogLevel Log配置转换为spanlog配置，默认不填的日志级别为warn
func getLogLevel(level string) int {
	switch level {
	case "all":
		return spanLog.AllLevel
	case "trace":
		return spanLog.TraceLevel
	case "debug":
		return spanLog.DebugLevel
	case "info":
		return spanLog.InfoLevel
	case "warn":
		return spanLog.WarnLevel
	case "error":
		return spanLog.ErrorLevel
	case "fatal":
		return spanLog.FatalLevel
	case "off":
		return spanLog.OffLevel
	default:
		return spanLog.WarnLevel
	}
}
