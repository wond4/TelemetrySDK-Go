package public

import (
	"testing"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/config"
	msqclient "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-mq-go"
)

var (
	client msqclient.ProtonMQClient
	err    error
)

func TestMain(m *testing.M) {
	config := &config.ProtonMqExporterTyp{
		Enable: true,
		Config: config.ProtonmqExporterConfig{
			SubType:   config.ProtonMqKafka,
			PubServer: "10.4.110.244",
			PubPort:   31000,
			Topic:     "proton-test",
			UserName:  "anyrobot",
			PassWord:  "eisoo.com123",
		},
	}
	client, err = initProtonMqClient(config)
	if err != nil {
		return
	}
	m.Run()
}

func TestNewProtonMqClient(t *testing.T) {

	msg := `[{"Link": {"TraceId": "00000000000000000000000000000000","SpanId": "0000000000000000"},"Timestamp": "2024-03-28T06:28:20.172550047Z","SeverityText": "Info","Body": {"Message": "AnyRobot Logger init success"},"Attributes": {},"Resource": {"host":{"arch":"x86_64","ip":"10.4.70.148","name":"ubuntu"},"os":{"description":"ubuntu","type":"linux","version":"22.04"},"service":{"instance":{"id":""},"name":"my-service-2","version":"UnknownServiceVersion"},"telemetry":{"sdk":{"language":"go","name":"TelemetrySDK-Go/exporter/ar_log","version":"2.7.5"}}}},{"Link": {"TraceId": "00000000000000000000000000000000","SpanId": "0000000000000000"},"Timestamp": "2024-03-28T06:28:20.172740878Z","SeverityText": "Info","Body": {"Message": "/root/GoProject/TelemetrySDK-Go/exporter/log/main.go:13:main: this is log"},"Attributes": {},"Resource": {"host":{"arch":"x86_64","ip":"10.4.70.148","name":"ubuntu"},"os":{"description":"ubuntu","type":"linux","version":"22.04"},"service":{"instance":{"id":""},"name":"my-service-2","version":"UnknownServiceVersion"},"telemetry":{"sdk":{"language":"go","name":"TelemetrySDK-Go/exporter/ar_log","version":"2.7.5"}}}}]`

	if err := client.Pub("proton-test", []byte(msg)); err != nil {
		t.Error(err)
		return
	}
	t.Log("done")
}
