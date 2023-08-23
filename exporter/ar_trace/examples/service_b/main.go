package main

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_trace/examples"
)

func main() {
	examples.StdoutTraceInit()
	//examples.ServerBefore()
	examples.Server()
	examples.TraceProviderExit(context.Background())
}
