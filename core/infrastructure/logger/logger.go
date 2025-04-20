package logger

import (
	"cland.org/cland-chat-service/core/infrastructure/config"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	once   sync.Once
	single *zap.Logger
	cfg    config.LogConfig
)

func GetLogger() *zap.Logger {
	once.Do(func() {
		var err error
		single, err = NewFromConfig(cfg)
		if err != nil {
			log.Fatalf("Failed to init logger: %v", err)
		}
		defer func() {
			if err := single.Sync(); err != nil {
				log.Printf("Failed to sync logger: %v", err)
			}
		}()
	})
	return single
}

func InitConfig(cfg0 config.LogConfig) {
	cfg = cfg0
}

// Config 定义了日志配置参数
type Config struct {
	Level      string `json:"level" yaml:"level"`           // 日志级别: debug, info, warn, error, dpanic, panic, fatal
	Filename   string `json:"filename" yaml:"filename"`     // 日志文件路径
	MaxSize    int    `json:"maxSize" yaml:"maxSize"`       // 单个日志文件最大大小(MB)
	MaxBackups int    `json:"maxBackups" yaml:"maxBackups"` // 保留的旧日志文件最大数量
	MaxAge     int    `json:"maxAge" yaml:"maxAge"`         // 保留旧日志文件的最大天数
	Compress   bool   `json:"compress" yaml:"compress"`     // 是否压缩/归档旧日志文件
}

func NewFromConfig(cfg config.LogConfig) (*zap.Logger, error) {
	loggerCfg := Config{
		Level:      cfg.Level,
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}
	return New(loggerCfg)
}

// New 创建并配置一个新的zap.Logger实例
func New(cfg Config) (*zap.Logger, error) {
	// 设置日志级别
	level := parseLogLevel(cfg.Level)

	// 配置日志输出
	writeSyncer := buildWriteSyncer(cfg)

	// 创建编码器
	encoder := buildEncoder()

	// 创建核心并构建logger
	core := zapcore.NewCore(encoder, writeSyncer, level)
	return buildLogger(core), nil
}

// parseLogLevel 解析日志级别字符串
func parseLogLevel(levelStr string) zapcore.Level {
	var level zapcore.Level
	if err := level.UnmarshalText([]byte(levelStr)); err != nil {
		return zapcore.InfoLevel // 默认级别
	}
	return level
}

// buildWriteSyncer 构建日志输出目标
func buildWriteSyncer(cfg Config) zapcore.WriteSyncer {
	// 文件日志输出
	fileWriter := &lumberjack.Logger{
		Filename:   cfg.Filename,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}

	// 多路输出: 控制台和文件
	return zapcore.NewMultiWriteSyncer(
		zapcore.AddSync(os.Stdout),  // 控制台输出
		zapcore.AddSync(fileWriter), // 文件输出
	)
}

// buildEncoder 构建日志编码器
func buildEncoder() zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseColorLevelEncoder, // 带颜色的级别显示
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.MillisDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	return zapcore.NewConsoleEncoder(encoderConfig) // 更友好的控制台输出格式
}

// buildLogger 构建最终的logger实例
func buildLogger(core zapcore.Core) *zap.Logger {
	return zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1), // 跳过包装函数的调用栈
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
}

// GinLogger 返回一个gin框架的日志中间件
func GinLogger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 收集日志字段
		fields := []zap.Field{
			zap.Int("status", c.Writer.Status()),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", c.ClientIP()),
			zap.String("user-agent", c.Request.UserAgent()),
			zap.Duration("latency", time.Since(start)),
		}

		// 添加错误信息(如果有)
		if len(c.Errors) > 0 {
			fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
		}

		// 根据状态码决定日志级别
		if c.Writer.Status() >= http.StatusInternalServerError {
			log.Error("server error", fields...)
		} else if c.Writer.Status() >= http.StatusBadRequest {
			log.Warn("client error", fields...)
		} else {
			log.Info("request completed", fields...)
		}
	}
}

// GinRecovery 返回一个gin框架的恢复中间件
func GinRecovery(log *zap.Logger, stack bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 检查是否是网络连接错误
				if isBrokenPipe(err) {
					logNetworkError(c, log, err)
					c.Error(err.(error))
					c.Abort()
					return
				}

				logPanic(c, log, err, stack)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
		}()
		c.Next()
	}
}

// isBrokenPipe 检查错误是否是网络连接错误
func isBrokenPipe(err interface{}) bool {
	ne, ok := err.(*net.OpError)
	if !ok {
		return false
	}
	se, ok := ne.Err.(*os.SyscallError)
	if !ok {
		return false
	}
	errStr := strings.ToLower(se.Error())
	return strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "connection reset by peer")
}

// logNetworkError 记录网络连接错误
func logNetworkError(c *gin.Context, log *zap.Logger, err interface{}) {
	httpRequest, _ := httputil.DumpRequest(c.Request, false)
	log.Error("broken pipe",
		zap.Any("error", err),
		zap.String("request", c.Request.Method+" "+c.Request.URL.Path),
		zap.String("request-headers", string(httpRequest)),
	)
}

// logPanic 记录panic信息
func logPanic(c *gin.Context, log *zap.Logger, err interface{}, stack bool) {
	httpRequest, _ := httputil.DumpRequest(c.Request, false)
	fields := []zap.Field{
		zap.Time("time", time.Now()),
		zap.Any("error", err),
		zap.String("request", c.Request.Method+" "+c.Request.URL.Path),
		zap.String("query", c.Request.URL.RawQuery),
		zap.String("ip", c.ClientIP()),
		zap.String("request-headers", string(httpRequest)),
	}

	if stack {
		fields = append(fields, zap.Stack("stack"))
	}

	log.Error("[Recovery from panic]", fields...)
}
