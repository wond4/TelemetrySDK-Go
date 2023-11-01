# 典型API服务可观测埋点示例

original目录下为没有可观测数据埋点的原始代码，observable目录下为添加可观测数据埋点之后的代码。\
该示例有两个golang服务。portal_service为入口服务，接受外部请求，然后调用dependent_service的接口。

## 运行代码示例，上报可观测数据到AnyRobot
1、下载go依赖
```shell
go mod tidy
```
2、设置环境变量,其中AnyRobot的IP地址和端口根据情况修改
```shell
export TELEMETRY_TRACE_ENDPOINT=10.4.104.243:80/api/feed_ingester/v1/jobs/job-a2491f67d02e482c/events
export TELEMETRY_TRACE_ENABLED=true
```
3、登录AnyRobot页面查看应用软件可观测仪表盘
