package ar_log

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/config"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/public"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/resource"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/encoder"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/exporter"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/field"
	spanLog "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/log"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/open_standard"
	sdkRuntime "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/runtime"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
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

// InitLogger 初始化上报到AnyRobot的程序日志记录器
// cfgType 配置类型，可选：cm、yaml
// cfgName 配置文件名称，如配置类型为cm，则该参数为configmap的名称；如配置类型为yaml，则该参数为yaml文件的路径，比如./ob-app.yaml
// serverName 微服务名称
func InitLogger(cfgType string, cfgName string, serverName string) {
	logConfig := &config.YamlLogConfig{
		Enabled:  "false",
		Endpoint: "",
		Level:    "",
	}
	Logger = initARLogger(logConfig, serverName)

	if cfgType == "cm" { // 如果配置为configmap形式
		// 监听configmap的内容，更新全局链路数据记录器的配置
		config.CmName = cfgName
		if kubeClient := config.InitKubeClient(); kubeClient != nil {
			watchConfigMap(kubeClient)
		}
	} else if cfgType == "yaml" { // 如果配置为yaml文件形式
		config.CfgFileNameLog = cfgName
		// 初始化配置
		config.NewLogConfig()
		Logger = initARLogger(config.YamlLogCfg, "")
		config.LogVP.OnConfigChange(func(e fsnotify.Event) {
			fmt.Printf("Log config file changed:%s, update logger\n", e)
			config.LoadLogConfig()
			Logger = initARLogger(config.YamlLogCfg, "")
		})
	}

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

// initARLogger 初始化上报到AnyRobot的日志记录器
// logEnabled 日志开关
// logEndpoint 日志上报地址，为空则打印标准输出
// logLevel 日志等级
// ServerName 微服务名称
func initARLogger(logConfig *config.YamlLogConfig, serverName string) spanLog.Logger {
	if logConfig == nil {
		return nil
	}

	var (
		logEnabled = logConfig.Enabled
		logLevel   = logConfig.Level
	)

	serverInstance := os.Getenv("HOSTNAME")
	if logEnabled != "true" {
		logLevel = "off"
	}

	// 初始化ar_log
	var ARLogger = spanLog.NewSamplerLogger(spanLog.WithSample(1.0), spanLog.WithLevel(getLogLevel(logLevel)))

	// 设置微服务相关信息
	if serverName != "" {
		resource.SetServiceName(serverName)
	}
	resource.SetServiceInstance(serverInstance)

	var systemLogWriter open_standard.Writer

	//初始化
	systemLogExporters := initExporters(logConfig)
	if len(systemLogExporters) <= 0 {
		ARLogger.Error(fmt.Sprintf("initARLogger 初始化initExporters数据有误，长度:%d", len(systemLogExporters)))
		return nil
	}
	systemLogWriter = open_standard.OpenTelemetryWriter(
		encoder.NewJsonEncoderWithExporters(systemLogExporters...),
		resource.LogResource())

	systemLogRunner := sdkRuntime.NewRuntime(systemLogWriter, field.NewSpanFromPool)
	systemLogRunner.SetUploadInternalAndMaxLog(3*time.Second, 10)

	go systemLogRunner.Run()
	ARLogger.SetLevel(getLogLevel(logLevel))
	ARLogger.SetRuntime(systemLogRunner)

	ARLogger.Info("AnyRobot Logger init success")

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

	businessLogger.Info("AnyRobot BLogger init success")
	return businessLogger
}

func watchConfigMap(client *kubernetes.Clientset) {
	configMapClient := client.CoreV1().ConfigMaps("")

	watcher, err := configMapClient.Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", config.CmName)})
	if err != nil {
		fmt.Printf("[TelemetrySDK]Failed to watch ConfigMaps: %+v\n", err.Error())
		return
	}

	fmt.Println("[TelemetrySDK]Starting to watch ConfigMaps...")

	go func() {
		var lc config.CmLogConfig
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				fmt.Printf("[TelemetrySDK]ConfigMap Added: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[config.CmMapKeyLog]), &lc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				fmt.Printf("[TelemetrySDK]Log Config Content: %+v\n", &lc)

				logConfig := &config.YamlLogConfig{
					Enabled:  config.GetLogEnabled(&lc),
					Endpoint: lc.Endpoint,
					Level:    lc.Level,
				}
				Logger = initARLogger(logConfig, "")
			case watch.Modified:
				fmt.Printf("[TelemetrySDK]ConfigMap Modified: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[config.CmMapKeyLog]), &lc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				fmt.Printf("[TelemetrySDK]Log Config Content: %+v\n", &lc)

				logConfig := &config.YamlLogConfig{
					Enabled:  config.GetLogEnabled(&lc),
					Endpoint: lc.Endpoint,
					Level:    lc.Level,
				}
				Logger = initARLogger(logConfig, "")
			case watch.Deleted:
				logConfig := &config.YamlLogConfig{
					Enabled:  "false",
					Endpoint: "",
					Level:    "",
				}
				fmt.Printf("[TelemetrySDK]ConfigMap Deleted: %s\n", event.Object.(*corev1.ConfigMap).Name)
				Logger = initARLogger(logConfig, "")
			}
		}
	}()

}

// initExportersFromEndpoint 兼容老版本endpont的配置
func initExportersFromEndpoint(logConfig *config.YamlLogConfig) []exporter.LogExporter {
	var (
		systemLogExporters []exporter.LogExporter
		logEndpoint        = logConfig.Endpoint
	)
	//老版本中 未配置endpoint 的表示输出到控制台，配置了endpoint表示输出到http
	if logEndpoint == "" {
		// 设置日志打印标准输出
		systemLogExporters = append(systemLogExporters, exporter.GetRealTimeExporter())
	} else {
		systemLogClient := public.NewHTTPClient(public.WithAnyRobotURL(logEndpoint),
			public.WithCompression(1),
			public.WithTimeout(10*time.Second),
			public.WithRetry(true, 5*time.Second, 20*time.Second, 1*time.Minute))

		systemLogExporters = append(systemLogExporters, NewExporter(systemLogClient))
	}
	return systemLogExporters
}

// initExporters 初始化Exporters
func initExporters(logConfig *config.YamlLogConfig) []exporter.LogExporter {
	var systemLogExporters []exporter.LogExporter

	//兼容老版本的配置
	if len(logConfig.Exporters) <= 0 {
		return initExportersFromEndpoint(logConfig)
	}
	//激活所有已启用的exporter
	for typ, typConfig := range logConfig.Exporters {
		var systemLogExporter exporter.LogExporter
		if !typConfig.Enable { //未开启
			continue
		}
		switch typ {
		case config.File:
			systemLogExporter = initFileExporter(logConfig)
		case config.Console:
			systemLogExporter = initConsoleExporter(logConfig)
		case config.Http:
			systemLogExporter = initHttpExporter(logConfig)
		case config.Mq:
			systemLogExporter = initProtonMqExporter(logConfig)
		default:
			Logger.Error(fmt.Sprintf("未知的exporter:%+v", typ))
		}
		if systemLogExporter == nil {
			continue
		}
		systemLogExporters = append(systemLogExporters, systemLogExporter)
	}
	return systemLogExporters
}

// initHttpExportersClient 2024-03-25 最新版本的配置
func initHttpExporter(logConfig *config.YamlLogConfig) exporter.LogExporter {
	// 设置日志通过HTTP上报
	var logEndpoint = logConfig.Exporters[config.Http].Config.HttpOutputConfig.Endpoint
	systemLogClient := public.NewHTTPClient(public.WithAnyRobotURL(logEndpoint),
		public.WithCompression(1),
		public.WithTimeout(10*time.Second),
		public.WithRetry(true, 5*time.Second, 20*time.Second, 1*time.Minute))
	return NewExporter(systemLogClient)
}

// initProtonMqExporter 初始化protonmq输出
func initProtonMqExporter(logConfig *config.YamlLogConfig) exporter.LogExporter {
	systemLogClient := public.NewProtonMqClient(logConfig.Exporters[config.Mq])
	return NewExporter(systemLogClient)
}

// initFileExporter 初始化文件输出
func initFileExporter(logConfig *config.YamlLogConfig) exporter.LogExporter {
	if logConfig == nil {
		return nil
	}
	stdoutPath := logConfig.Exporters[config.File].Config.FileOutputConfig.Path
	systemLogClient := public.NewFileClient(stdoutPath)
	return NewExporter(systemLogClient)
}

// initConsoleExporter 初始化console输出
func initConsoleExporter(logConfig *config.YamlLogConfig) exporter.LogExporter {
	return exporter.GetRealTimeExporter()
}
