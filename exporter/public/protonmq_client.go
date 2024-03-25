package public

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/config"
	msqclient "devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/proton-mq-go"
	"errors"
	"log"
)

// ProtonMqClient 客户端结构体。
type ProtonMqClient struct {
	stopCh chan struct{}
	client msqclient.ProtonMQClient
	cfg    *config.ProtonMqOutputConfig
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
		topic = c.cfg.Topic
	)
	if len(topic) == 0 {
		return errors.New("")
	}

	if err := c.client.Pub(topic, data); err != nil {
		return err
	}
	return nil
}

// NewProtonMqClient 创建Exporter的控制台+本地文件发送客户端。
func NewProtonMqClient(exportersTypConfig *config.ExportersTypConfig) Client {
	if exportersTypConfig == nil || exportersTypConfig.Config.ProtonMqOutputConfig == nil {
		return nil
	}
	var protonMqOutputConfig = exportersTypConfig.Config.ProtonMqOutputConfig
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
		log.Fatal("failed to create a proton mq client:", err)
		return nil
	}
	return &ProtonMqClient{client: client, cfg: protonMqOutputConfig}
}
