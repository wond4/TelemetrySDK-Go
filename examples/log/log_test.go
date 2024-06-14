package examplelog

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_log"
	"testing"
)

func Benchmark_InfoWithProton(b *testing.B) {
	ar_log.InitLogger("yaml", "log-sdk-config-with-proton", "my-service-2")

	for n := 0; n < b.N; n++ {
		ar_log.Info(context.Background(), "this is log")
	}
}

func Benchmark_InfoWithFile(b *testing.B) {
	ar_log.InitLogger("yaml", "log-sdk-config-with-file", "my-service-2")

	for n := 0; n < b.N; n++ {
		ar_log.Info(context.Background(), "this is log")
	}
}

func Test_send(t *testing.T) {
	ar_log.InitLogger("yaml", "log-sdk-config-with-file", "my-service-2")
	for i := 0; i < 1; i++ {
		ar_log.Info(context.Background(), "this is log")
	}
}
