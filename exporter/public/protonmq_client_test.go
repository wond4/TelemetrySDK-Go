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
	if err := client.Pub("proton-test", []byte("test-send")); err != nil {
		t.Error(err)
		return
	}
	t.Log("done")
}
