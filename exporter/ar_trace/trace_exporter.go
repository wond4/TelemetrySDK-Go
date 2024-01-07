package ar_trace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/common"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/config"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/public"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/resource"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/version"
	"github.com/fsnotify/fsnotify"
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
)

// 跨包实现接口占位用。
var _ sdktrace.SpanExporter = (*TraceExporter)(nil)

// Tracer 是一个全局变量，用于在业务代码中生产Span。
var Tracer = otel.GetTracerProvider().Tracer(
	version.TraceInstrumentationName,
	trace.WithInstrumentationVersion(version.TelemetrySDKVersion),
	trace.WithSchemaURL(version.TraceInstrumentationURL),
)

func updateTracer() {
	Tracer = otel.GetTracerProvider().Tracer(
		version.TraceInstrumentationName,
		trace.WithInstrumentationVersion(version.TelemetrySDKVersion),
		trace.WithSchemaURL(version.TraceInstrumentationURL),
	)
}

var tp = (*sdktrace.TracerProvider)(nil)
var te = &TraceExporter{}

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

// InitTracer 初始化函数，初始化全局链路数据记录器
// cfgType 配置类型，可选：cm、yaml
// cfgName 配置文件名称，如配置类型为cm，则该参数为configmap的名称；如配置类型为yaml，则该参数为yaml文件的路径，比如./ob-app.yaml
// serverName 微服务名称
func InitTracer(cfgType string, cfgName string, serverName string) {
	// 先初始化一个不记录链路数据的全局链路数据记录器
	InitSilentTracer(serverName)
	if cfgType == "cm" { // 如果配置为configmap形式
		// 监听configmap的内容，更新全局链路数据记录器的配置
		config.CmName = cfgName
		if kubeClient := config.InitKubeClient(); kubeClient != nil {
			watchConfigMap(kubeClient)
		}
	} else if cfgType == "yaml" { // 如果配置为yaml文件形式
		config.CfgFileNameTrace = cfgName
		// 初始化配置
		config.NewTraceConfig()
		UpdateTracerClient(config.YamlTraceCfg.Enabled, config.YamlTraceCfg.Endpoint)
		config.TraceVP.OnConfigChange(func(e fsnotify.Event) {
			fmt.Printf("Trace config file changed:%s, update tracer client\n", e)
			config.LoadTraceConfig()
			UpdateTracerClient(config.YamlTraceCfg.Enabled, config.YamlTraceCfg.Endpoint)
		})
	}

}

// InitSilentTracer 初始化全局链路数据记录器，不记录数据
func InitSilentTracer(serverName string) {
	serverInstance := os.Getenv("HOSTNAME")

	if serverName != "" {
		resource.SetServiceName(serverName)
	}
	resource.SetServiceInstance(serverInstance)

	traceClient := public.NewSilentClient()
	te = NewExporter(traceClient)
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(te,
			sdktrace.WithMaxExportBatchSize(1000)),
		sdktrace.WithResource(TraceResource()))

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	updateTracer()
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

func watchConfigMap(clientset *kubernetes.Clientset) {
	configMapClient := clientset.CoreV1().ConfigMaps("")

	watcher, err := configMapClient.Watch(context.TODO(), metav1.ListOptions{FieldSelector: fmt.Sprintf("metadata.name=%s", config.CmName)})
	if err != nil {
		fmt.Printf("[TelemetrySDK]Failed to watch ConfigMaps: %+v\n", err.Error())
		return
	}

	fmt.Println("[TelemetrySDK]Starting to watch ConfigMaps...")

	go func() {
		var tc config.CmTraceConfig
		for event := range watcher.ResultChan() {
			switch event.Type {
			case watch.Added:
				fmt.Printf("[TelemetrySDK]ConfigMap Added: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[config.CmMapKeyTrace]), &tc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				fmt.Printf("[TelemetrySDK]Trace Config Content: %+v\n", &tc)

				UpdateTracerClient(config.GetTraceEnabled(&tc), tc.Endpoint)
			case watch.Modified:
				fmt.Printf("[TelemetrySDK]ConfigMap Modified: %s\n", event.Object.(*corev1.ConfigMap).Name)

				err := yaml.Unmarshal([]byte(event.Object.(*corev1.ConfigMap).Data[config.CmMapKeyTrace]), &tc)
				if err != nil {
					fmt.Printf("[TelemetrySDK]error: %v", err)
				}

				fmt.Printf("[TelemetrySDK]Trace Config Content: %+v\n", &tc)

				UpdateTracerClient(config.GetTraceEnabled(&tc), tc.Endpoint)
			case watch.Deleted:
				fmt.Printf("[TelemetrySDK]ConfigMap Deleted: %s\n", event.Object.(*corev1.ConfigMap).Name)
				UpdateTracerClient("false", "")
			}
		}
	}()

}
