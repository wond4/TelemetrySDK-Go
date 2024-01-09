package main

import (
	"context"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_log"
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_trace"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"net/http"
)

type User struct {
	ID   uint
	Name string
	Age  int
}

func main() {
	// 通过configmap设置配置，适用于k8s集群内部应用
	ar_trace.InitTracer("cm", "anyshare-telemetry-sdk", "my-service-2")
	// 通过yaml文件设置配置，适用于k8s集群外部应用
	//ar_trace.InitTracer("yaml", "ob-app-config-trace", "my-service-2")
	defer ar_trace.ShutdownTracer()
	// 通过configmap设置配置，适用于k8s集群内部应用
	ar_log.InitLogger("cm", "anyshare-telemetry-sdk", "my-service-2")
	// 通过yaml文件设置配置，适用于k8s集群外部应用
	//ar_log.InitLogger("yaml", "ob-app-config-log", "my-service-2")
	initDB()

	r := gin.Default()
	r.Use(otelgin.Middleware("my-server-dependent"))
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		user, _ := getUser(id, c)
		c.String(http.StatusOK, user.Name)
	})
	_ = r.Run(":50081")
}

// getUser 根据用户ID获取用户信息
func getUser(id string, ctx context.Context) (result User, err error) {
	ctx, span := ar_trace.StartInternalSpan(ctx)
	// 结束span时，err如果不为空的话，则span状态设置为error
	defer func() { ar_trace.EndSpan(ctx, err) }()
	// 不设置span name的话，span name 默认为函数名称
	span.SetName("根据用户ID获取用户信息")

	// 连接到 SQLite 数据库
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败：" + err.Error())
	}
	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		panic("连接数据库失败：" + err.Error())
	}

	// WHERE 查询
	err = db.WithContext(ctx).Where("id = ?", id).First(&result).Error
	if err != nil {
		fmt.Println("查询用户失败：" + err.Error())
	}
	return
}

// initDB 初始化本地数据库
func initDB() {
	// 连接到 SQLite 数据库
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败：" + err.Error())
	}

	// 创建数据表
	err = db.AutoMigrate(&User{})
	if err != nil {
		panic("创建数据表失败：" + err.Error())
	}

	// 创建用户
	user := User{Name: "张三", Age: 30, ID: 1}

	// 查询是否已存在记录
	result := db.Where("id = ?", user.ID).First(&user)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		panic("查询记录失败：" + result.Error.Error())
	}

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		// 不存在记录，插入数据
		result = db.Create(&user)
		if result.Error != nil {
			panic("插入数据失败：" + result.Error.Error())
		}
		fmt.Println("数据插入成功")
	} else {
		fmt.Println("数据已存在")
	}
}
