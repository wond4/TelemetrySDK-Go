package main

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_log"
	"testing"
)

func BenchmarkInfo(b *testing.B) {
	ar_log.InitLogger("yaml", "log-sdk-config", "my-service-2")

	for n := 0; n < b.N; n++ {
		ar_log.Info(context.Background(), "this is log")
	}
}
