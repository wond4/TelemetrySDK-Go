package ar_trace

import (
	"bytes"
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/common"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/public"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/resource"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/version"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	sdkresource "go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode"
)

// 跨包实现接口占位用。
var _ sdktrace.SpanExporter = (*TraceExporter)(nil)

// Tracer 是一个全局变量，用于在业务代码中生产Span。
var Tracer = otel.GetTracerProvider().Tracer(
	version.TraceInstrumentationName,
	trace.WithInstrumentationVersion(version.TelemetrySDKVersion),
	trace.WithSchemaURL(version.TraceInstrumentationURL),
)

var tp = (*sdktrace.TracerProvider)(nil)
var te = &TraceExporter{}

// configmap相关配置
var cmNamespace = "anyrobot"
var cmName = "cnao-aso-cm"
var cmMapKeyLog = "trace-sdk-config.yaml"

// TraceConfig 链路数据记录器配置，结构体映射到YAML数据结构
type TraceConfig struct {
	Enabled       string   `yaml:"enabled"`
	Endpoint      string   `yaml:"endpoint"`
	EnabledAllPod string   `yaml:"enabledAllPod"`
	EnabledPods   []string `yaml:"enabledPods"`
}

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

// init 包初始化函数，初始化全局链路数据记录器
func init() {
	// 先初始化一个不记录链路数据的全局链路数据记录器
	InitSilentTracer()
	// 监听configmap的内容，更新全局链路数据记录器的配置
	if kubeClient := initKubeClient(); kubeClient != nil {
		watchConfigMap(kubeClient)
	}
}

// InitSilentTracer 初始化全局链路数据记录器，不记录数据
func InitSilentTracer() {
	serverName := os.Getenv("TELEMETRY_SERVICE_NAME")
	serverVersion := os.Getenv("TELEMETRY_SERVICE_VERSION")
	serverInstance := os.Getenv("HOSTNAME")

	resource.SetServiceName(serverName)
	resource.SetServiceVersion(serverVersion)
	resource.SetServiceInstance(serverInstance)

	traceClient := public.NewSilentClient()
	te = NewExporter(traceClient)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(te,
			sdktrace.WithMaxExportBatchSize(1000)),
		sdktrace.WithResource(TraceResource()))

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
}

// UpdateTracerClient 更新全局链路数据记录器的数据发送客户端
func UpdateTracerClient(traceEnabled string, traceEndpoint string) {
	if traceEnabled == "true" && traceEndpoint != "" {
		traceClient := public.NewHTTPClient(public.WithAnyRobotURL(traceEndpoint),
			public.WithCompression(1), public.WithTimeout(10*time.Second),
			public.WithRetry(true, 5*time.Second, 30*time.Second, 1*time.Minute))
		StopTracerClient()
		te.SetClient(traceClient)
	} else if traceEnabled == "true" && traceEndpoint == "" {
		traceClient := public.NewConsoleClient()
		StopTracerClient()
		te.SetClient(traceClient)
	} else {
		traceClient := public.NewSilentClient()
		StopTracerClient()
		te.SetClient(traceClient)
	}
}

// StopTracerClient 关闭全局链路数据记录器的数据发送客户端
func StopTracerClient() {
	// 关闭旧的client
	// 设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := te.GetClient().Stop(ctx)
	if err != nil {
		log.Printf("[TelemetrySDK]Error shutting down tracer client: %v", err)
	}
}

// GetTracer 获取全局链路数据记录器
func GetTracer() *sdktrace.TracerProvider {
	return tp
}

// ShutdownTracer 关闭全局链路数据记录器
func ShutdownTracer() {
	if tp == nil {
		return
	}

	// 设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := tp.Shutdown(ctx); err != nil {
		log.Printf("[TelemetrySDK]Error shutting down tracer provider: %v", err)
	}

	tp = nil
}

// InitARTracer 初始化上报到AnyRobot的链路数据记录器
// ServerName 微服务名称
// ServerVersion 微服务版本
// ServerInstance 微服务实例标识
func InitARTracer() *sdktrace.TracerProvider {
	serverName := os.Getenv("TELEMETRY_SERVICE_NAME")
	serverVersion := os.Getenv("TELEMETRY_SERVICE_VERSION")
	serverInstance := os.Getenv("HOSTNAME")
	traceEnabled := os.Getenv("TELEMETRY_TRACE_ENABLED")
	traceUrl := os.Getenv("TELEMETRY_TRACE_ENDPOINT")

	if traceEnabled == "true" {
		resource.SetServiceName(serverName)
		resource.SetServiceVersion(serverVersion)
		resource.SetServiceInstance(serverInstance)

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
func StopARTracer(tp *sdktrace.TracerProvider) {
	if tp == nil {
		return
	}

	// 设置超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := tp.Shutdown(ctx); err != nil {
		log.Printf("[TelemetrySDK]Error shutting down tracer provider: %v", err)
	}
}

// StartInternalSpan 内部方法调用trace埋点
func StartInternalSpan(ctx context.Context) (context.Context, trace.Span) {
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
	}

	pc, file, linkNo, ok := runtime.Caller(1)
	if !ok {
		log.Printf("[TelemetrySDK]start span error")
		ctx, span := Tracer.Start(ctx, "unKnow", trace.WithSpanKind(trace.SpanKindInternal))
		return ctx, span
	} else {
		funcPaths := strings.Split(runtime.FuncForPC(pc).Name(), "/")
		spanName := funcPaths[len(funcPaths)-1]
		ctx, span := Tracer.Start(ctx, spanName, trace.WithSpanKind(trace.SpanKindInternal))
		span.SetAttributes(attribute.String("func.path", fmt.Sprintf("%s:%v", file, linkNo)))
		return ctx, span
	}
}

// EndSpan 关闭span
func EndSpan(ctx context.Context, err error) {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "OK")
	}
	span.End()
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

func watchConfigMap(clientset *kubernetes.Clientset) {
	configMapClient := clientset.CoreV1().ConfigMaps(cmNamespace)

	watcher, err := configMapClient.Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", cmName)})
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("[TelemetrySDK]Starting to watch ConfigMaps...")

	go func() {
		var tc TraceConfig
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				fmt.Printf("[TelemetrySDK]ConfigMap Added: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[cmMapKeyLog]), &tc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				UpdateTracerClient(getTraceEnabled(&tc), tc.Endpoint)
			case watch.Modified:
				fmt.Printf("[TelemetrySDK]ConfigMap Modified: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[cmMapKeyLog]), &tc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				UpdateTracerClient(getTraceEnabled(&tc), tc.Endpoint)
			case watch.Deleted:
				fmt.Printf("[TelemetrySDK]ConfigMap Deleted: %s\n", event.Object.(*corev1.ConfigMap).Name)
				UpdateTracerClient("false", "")
			}
		}
	}()

}

// getTraceEnabled 获取链路数据记录器开关配置。如果功能开关为false，则不开启链路数据记录；反之，如果所有微服务开关为true，则开启链路数据记录；
// 反之，则判断pod名称前缀是否在配置中，如在则开启链路数据记录。
func getTraceEnabled(tc *TraceConfig) string {
	if tc.Enabled == "true" {
		if tc.EnabledAllPod == "true" {
			return "true"
		} else {
			for _, item := range tc.EnabledPods {
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
