package main

import (
	"context"
	"time"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_log"
)

func main() {
	initSDK() //这是初始化
	send()    //发送内容

	time.Sleep(time.Second * 5)
}

// 这是初始化的
func initSDK() {
	ar_log.InitLogger("yaml", "ob-app-config-log", "my-service-2")

}

// 这是发送内容的
func send() {
	for i := 0; i < 1; i++ {
		ar_log.Info(context.Background(), "this is log")
	}
}
