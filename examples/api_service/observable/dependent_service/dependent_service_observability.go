package main

import (
	"devops.aishu.cn/AISHUDevOps/ONE-Architecture/_git/TelemetrySDK-Go.git/exporter/v2/ar_trace"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
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
	arTracerProvider := ar_trace.InitARTracer()
	defer ar_trace.StopARTracer(arTracerProvider)

	initDB()

	r := gin.Default()
	r.Use(otelgin.Middleware("my-server-dependent"))
	r.GET("/users/:id", func(c *gin.Context) {
		id := c.Param("id")
		user := getUser(id, c)
		c.String(http.StatusOK, user.Name)
	})
	_ = r.Run(":50081")
}

// getUser 根据用户ID获取用户
func getUser(id string, c *gin.Context) User {
	// 第二个参数为span名称、第三个参数为span类型
	ctx, span := ar_trace.Tracer.Start(c.Request.Context(), "根据用户ID获取用户", trace.WithSpanKind(trace.SpanKindInternal))
	defer span.End()
	// 第一个参数为span状态、第二个参数为span状态描述
	span.SetStatus(codes.Ok, "")

	// 连接到 SQLite 数据库
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		panic("连接数据库失败：" + err.Error())
	}
	if err := db.Use(otelgorm.NewPlugin()); err != nil {
		panic("连接数据库失败：" + err.Error())
	}

	// WHERE 查询
	var result User
	err = db.WithContext(ctx).Where("id = ?", id).First(&result).Error
	if err != nil {
		fmt.Println("查询用户失败：" + err.Error())
	}
	return result
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
