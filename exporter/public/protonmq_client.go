package public

import (
	"context"

	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/config"
	msqclient "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-mq-go"
	"github.com/pkg/errors"
)

// ProtonMqClient 客户端结构体。
type ProtonMqClient struct {
	stopCh chan struct{}
	client msqclient.ProtonMQClient
	cfg    *config.ProtonMqExporterTyp
}

// Path 获取上报地址。
func (c *ProtonMqClient) Path() string {
	return ""
}

// Stop 关闭发送器。
func (c *ProtonMqClient) Stop(ctx context.Context) error {
	close(c.stopCh)
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

// UploadData 批量发送可观测性数据。
func (c *ProtonMqClient) UploadData(ctx context.Context, data []byte) error {
	// 退出逻辑关闭了发送。
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.stopCh:
		return nil
	default:

	}
	var (
		topic = c.cfg.Config.Topic
	)
	if len(topic) == 0 {
		return errors.New("未配置正确的Topic")
	}

	if err := c.Pub(topic, data); err != nil {
		return err
	}
	return nil
}

func (c *ProtonMqClient) Pub(topic string, data []byte) error {
	if err := c.client.Pub(topic, data); err != nil {
		return err
	}
	return nil
}

// NewProtonMqClient 创建Exporter的控制台+本地文件发送客户端。
func NewProtonMqClient(config *config.ProtonMqExporterTyp) Client {
	if config == nil || !config.Enable {
		return nil
	}
	client, err := initProtonMqClient(config)
	if err != nil {
		return nil
	}
	return &ProtonMqClient{cfg: config, client: client}
}

func initProtonMqClient(config *config.ProtonMqExporterTyp) (msqclient.ProtonMQClient, error) {
	var protonMqOutputConfig = config.Config
	var (
		username  = protonMqOutputConfig.UserName
		password  = protonMqOutputConfig.PassWord
		pubServer = protonMqOutputConfig.PubServer
		pubPort   = protonMqOutputConfig.PubPort
		subServer = protonMqOutputConfig.SubServer
		subPort   = protonMqOutputConfig.SubPort
	)
	opts := []msqclient.ClientOpt{
		msqclient.UserInfo(username, password),
		msqclient.AuthMechanism("PLAIN"),
	}

	client, err := msqclient.NewProtonMQClient(pubServer, pubPort, subServer, subPort, protonMqOutputConfig.SubType.String(), opts...)

	if err != nil {
		return nil, errors.Errorf("failed to create a proton mq client: %+v", err)
	}
	return client, nil
}
