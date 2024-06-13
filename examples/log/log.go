package main

import (
	"context"
	"time"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_log"
)

func main() {
	ar_log.InitLogger("yaml", "log-sdk-config", "my-service-2")

	ar_log.Info(context.Background(), "this is log")

	time.Sleep(time.Second * 5)
}
