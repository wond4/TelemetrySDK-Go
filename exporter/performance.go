package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/ar_trace"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/public"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	setTrace()
}
func setTrace() {
	wg := sync.WaitGroup{}
	ctx := context.Background()
	attrs := []attribute.KeyValue{
		attribute.String("job_id", "test-multipipelines"),
	}
	jobResource := resource.NewWithAttributes("", attrs...)
	tempResource, err := resource.Merge(jobResource, ar_trace.TraceResource())
	if err == nil {
		jobResource = tempResource
	}
	//traceExporter, _ := otlptracegrpc.New(ctx, otlptracegrpc.WithInsecure(), otlptracegrpc.WithEndpoint("10.4.68.236:30013"))
	traceClient := public.NewStdoutClient("./AnyRobotTrace.json")
	traceExporter := ar_trace.NewExporter(traceClient)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter, sdktrace.WithMaxQueueSize(500000),
			sdktrace.WithMaxExportBatchSize(500),
			sdktrace.WithExportTimeout(time.Hour)),
		sdktrace.WithResource(jobResource),
		sdktrace.WithSampler(
			sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.2))),
	)
	otel.SetTracerProvider(tracerProvider)
	defer func() {
		if err := tracerProvider.Shutdown(ctx); err != nil {
			log.Println(err)
		}
	}()
	startT := time.Now()
	log.Println("开始时间：" + startT.GoString())
	for i := 0; i < 1; i++ {
		wg.Add(1)
		go func() {
			for j := 1; j < 2; j++ {
				fmt.Println(j)
				multiply(ctx, 2, 3)
			}
			wg.Done()
		}()
	}
	wg.Wait()
	tc := time.Since(startT)
	log.Printf("结束时间：" + tc.String())
}

// multiply 增加了Trace埋点的计算两数之积。
func multiply(ctx context.Context, x, y int64) (context.Context, int64) {
	NewTraceState := trace.TraceState{}
	NewTraceState, _ = NewTraceState.Insert("congo", "t61rcWkgMzE")
	NewTraceState, _ = NewTraceState.Insert("rojo", "00f067aa0ba902b7")
	ctx = trace.ContextWithSpanContext(ctx, trace.NewSpanContext(trace.SpanContextConfig{TraceState: NewTraceState}))
	ctx, span := ar_trace.Tracer.Start(ctx, "乘法", trace.WithSpanKind(1))
	defer span.End()
	span.SetAttributes(attribute.KeyValue{Key: "multiply", Value: attribute.StringValue("计算两数之积")}, attribute.String("job_id", "test-multipipelines"))
	span.AddEvent("multiplyEvent", trace.WithAttributes(attribute.BoolSlice("key", []bool{true, true}), attribute.String("analyzed", "100ms")))
	span.SetStatus(2, "成功计算乘积")
	span.TracerProvider()
	//业务代码
	time.Sleep(100 * time.Millisecond)
	test(ctx)
	test(ctx)
	test(ctx)
	test(ctx)
	test(ctx)
	test(ctx)
	test(ctx)
	return ctx, x * y
}

func test(ctx context.Context) context.Context {
	ctx, span := ar_trace.Tracer.Start(ctx, "测试span", trace.WithSpanKind(1))
	span.SetAttributes(attribute.KeyValue{Key: "multiply", Value: attribute.StringValue("测试中")}, attribute.String("job_id", "test-multipipelines"))
	span.AddEvent("multiplyEvent", trace.WithAttributes(attribute.BoolSlice("key", []bool{true, true}), attribute.String("analyzed", "100ms")))
	span.SetStatus(2, "测试完成")
	defer span.End()
	return ctx
}
