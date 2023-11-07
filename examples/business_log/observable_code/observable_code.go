package main

import (
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_log"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/span/v2/field"
	"fmt"
	"time"
)

func main() {
	fmt.Println("hello world")

	bl := ar_log.InitBusinessLogger()
	msg := make(map[string]interface{})
	msg["文件名称"] = "文件A"
	bl.InfoField(field.MallocJsonField(msg), "数据产品元数据")
	// 日志打印有延迟，等待一会儿
	time.Sleep(5 * time.Second)
}
