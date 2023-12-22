# 典型API服务可观测埋点示例

original目录下为没有可观测数据埋点的原始代码，observable目录下为添加可观测数据埋点之后的代码。\
该示例有两个golang服务。portal_service为入口服务，接受外部请求，然后调用dependent_service的接口。

## trace埋点场景
1、gin server
2、http client
3、gorm client
更多场景可参考 https://opentelemetry.io/ecosystem/registry/?s=&component=instrumentation&language=go

## 运行代码示例，上报可观测数据到AnyRobot
1、下载go依赖，进入本文件所在目录执行下面命令
```shell
go mod tidy
```
2、设置环境变量,其中AnyRobot的IP地址和端口根据情况修改。TELEMETRY_TRACE_ENDPOINT为空时，链路数据会打印，用于调试。
```shell
export TELEMETRY_TRACE_ENDPOINT=http://10.4.104.243:80/api/feed_ingester/v1/jobs/job-a2491f67d02e482c/events
export TELEMETRY_TRACE_ENABLED=true
```
3、运行两个golang服务
```shell
go run observable/dependent_service/dependent_service_observability.go
go run observable/portal_service/portal_service_observability.go
```
4、外部请求
```shell
curl 127.0.0.1:50080/users/1
curl 127.0.0.1:50080/users/2
```
5、登录AnyRobot页面查看应用软件可观测仪表盘
