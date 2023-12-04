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
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
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

// configmap相关配置
var cmNamespace = "anyrobot"
var cmName = "cnao-aso-cm"
var cmMapKeyLog = "log-sdk-config.yaml"

// LogConfig 程序日志记录器配置，结构体映射到YAML数据结构
type LogConfig struct {
	Enabled       string   `yaml:"enabled"`
	Level         string   `yaml:"level"`
	EnabledAllPod string   `yaml:"enabledAllPod"`
	EnabledPods   []string `yaml:"enabledPods"`
}

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
	Logger = InitARLogger("false", "")
	BLogger = InitBusinessLogger()

	if kubeClient := initKubeClient(); kubeClient != nil {
		watchConfigMap(kubeClient)
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

// InitARLogger 初始化上报到AnyRobot的日志记录器
// ServerName 微服务名称
// ServerVersion 微服务版本
// ServerInstance 微服务实例标识
// logLevel 日志等级
func InitARLogger(logEnabled string, logLevel string) spanLog.Logger {
	serverName := os.Getenv("TELEMETRY_SERVICE_NAME")
	serverVersion := os.Getenv("TELEMETRY_SERVICE_VERSION")
	serverInstance := os.Getenv("HOSTNAME")
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

func initKubeClient() *kubernetes.Clientset {
	defer func() {
		if err := recover(); err != nil {
			fmt.Printf("[TelemetrySDK]创建kubernetes api客户端失败：%v\n", err)
		}
	}()

	// 使用Pod内的Service Account来创建一个kubernetes api客户端
	config, err := rest.InClusterConfig()
	if err != nil {
		fmt.Printf("[TelemetrySDK]在kubernetes集群主机创建kubernetes api客户端\n")
		// 当在集群外部调试时，使用kubeconfig文件
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Printf("[TelemetrySDK]在kubernetes集群主机创建kubernetes api客户端失败：%v\n", err.Error())
		}
	} else {
		fmt.Printf("[TelemetrySDK]在kubernetes集群内部创建kubernetes api客户端\n")
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Printf("[TelemetrySDK]创建kubernetes api客户端失败：%v\n", err.Error())
	}

	return client
}

func watchConfigMap(client *kubernetes.Clientset) {
	configMapClient := client.CoreV1().ConfigMaps(cmNamespace)

	watcher, err := configMapClient.Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", cmName)})
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("[TelemetrySDK]Starting to watch ConfigMaps...")

	go func() {
		var lc LogConfig
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				fmt.Printf("[TelemetrySDK]ConfigMap Added: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[cmMapKeyLog]), &lc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				Logger = InitARLogger(getLogEnabled(&lc), lc.Level)
			case watch.Modified:
				fmt.Printf("[TelemetrySDK]ConfigMap Modified: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[cmMapKeyLog]), &lc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				Logger = InitARLogger(getLogEnabled(&lc), lc.Level)
			case watch.Deleted:
				fmt.Printf("[TelemetrySDK]ConfigMap Deleted: %s\n", event.Object.(*corev1.ConfigMap).Name)
				Logger = InitARLogger("false", "")
			}
		}
	}()

}

// getLogEnabled 获取日志记录器开关配置。如果功能开关为false，则不开启日志记录；反之，如果所有微服务开关为true，则开启日志记录；
// 反之，则判断pod名称前缀是否在配置中，如在则开启日志记录。
func getLogEnabled(lc *LogConfig) string {
	if lc.Enabled == "true" {
		if lc.EnabledAllPod == "true" {
			return "true"
		} else {
			for _, item := range lc.EnabledPods {
				if podName := os.Getenv("HOSTNAME"); getPodNamePrefix(podName) == item {
					return "true"
				}
			}
		}
	}
	return "false"
}

// getPodNamePrefix 获取pod名称前面不变的部分
func getPodNamePrefix(podName string) string {
	if lastIndex := strings.LastIndex(podName, "-"); lastIndex != -1 {
		if isAllDigits(podName[lastIndex+1:]) {
			return podName[:lastIndex]
		} else {
			if last2Index := strings.LastIndex(podName[:lastIndex], "-"); last2Index != -1 {
				return podName[:last2Index]
			} else {
				return podName[:lastIndex]
			}
		}
	} else {
		return podName
	}
}

// isAllDigits 检查字符串是否全部由数字组成。
func isAllDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
