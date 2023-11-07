package ar_log

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/public"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/resource"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/encoder"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/exporter"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/field"
	spanLog "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/log"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/open_standard"
	sdkRuntime "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/runtime"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// 跨包实现接口占位用。
var _ exporter.LogExporter = (*SpanExporter)(nil)
var _ exporter.SyncExporter = (*syncExporter)(nil)
var (
	// Logger 全局程序日志记录器
	Logger spanLog.Logger
	// BLogger 全局业务日志记录器
	BLogger spanLog.Logger
)

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

// init 包初始化函数，初始化全局日志记录器
func init() {
	Logger = InitARLogger()
	BLogger = InitBusinessLogger()
	Logger.Info("AnyRobot Logger init success")
	BLogger.Info("AnyRobot BLogger init success")
}

// Debug 拼接上文件、行号、函数名。用于日志记录时把位置信息带上
func Debug(ctx context.Context, msg string) {
	pc, filename, line, _ := runtime.Caller(1)
	Logger.Debug(fmt.Sprintf("%s:%d:%s: %v", filename, line, strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(pc).Name()), "."), msg),
		field.WithContext(ctx))
}

func Info(ctx context.Context, msg string) {
	pc, filename, line, _ := runtime.Caller(1)
	Logger.Info(fmt.Sprintf("%s:%d:%s: %v", filename, line, strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(pc).Name()), "."), msg),
		field.WithContext(ctx))
}

func Warn(ctx context.Context, msg string) {
	pc, filename, line, _ := runtime.Caller(1)
	Logger.Warn(fmt.Sprintf("%s:%d:%s: %v", filename, line, strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(pc).Name()), "."), msg),
		field.WithContext(ctx))
}

func Error(ctx context.Context, msg string) {
	pc, filename, line, _ := runtime.Caller(1)
	Logger.Error(fmt.Sprintf("%s:%d:%s: %v", filename, line, strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(pc).Name()), "."), msg),
		field.WithContext(ctx))
}

func Fatal(ctx context.Context, msg string) {
	pc, filename, line, _ := runtime.Caller(1)
	Logger.Fatal(fmt.Sprintf("%s:%d:%s: %v", filename, line, strings.TrimPrefix(filepath.Ext(runtime.FuncForPC(pc).Name()), "."), msg),
		field.WithContext(ctx))
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
	systemLogRunner := sdkRuntime.NewRuntime(systemLogWriter, field.NewSpanFromPool)
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

// InitBusinessLogger 初始化业务日志记录器
func InitBusinessLogger() spanLog.Logger {
	// 设置微服务相关信息
	resource.SetServiceName(os.Getenv("TELEMETRY_SERVICE_NAME"))
	resource.SetServiceVersion(os.Getenv("TELEMETRY_SERVICE_VERSION"))
	resource.SetServiceInstance(os.Getenv("HOSTNAME"))

	var businessLogger = spanLog.NewSamplerLogger(spanLog.WithSample(1.0), spanLog.WithLevel(spanLog.AllLevel))
	systemLogExporter := exporter.GetRealTimeExporter()
	systemLogWriter := open_standard.OpenTelemetryWriter(
		encoder.NewJsonEncoderWithExporters(systemLogExporter),
		resource.LogResource())
	systemLogRunner := sdkRuntime.NewRuntime(systemLogWriter, field.NewSpanFromPool)
	systemLogRunner.SetUploadInternalAndMaxLog(3*time.Second, 10)
	// 运行SystemLogger日志器。
	go systemLogRunner.Run()
	businessLogger.SetRuntime(systemLogRunner)

	return businessLogger
}
