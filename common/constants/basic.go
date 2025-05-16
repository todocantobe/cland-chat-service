package constants

const (

	// API 相关
	APIBaseURL = "https://api.example.com"
	APITimeout = 30 // 秒

	// 数据库配置
	DBHost = "localhost"
	DBPort = 5432
	DBName = "mydb"

	// 状态码
	StatusSuccess = 200
	StatusError   = 500

	//业务相关
	KEY_USER_ID = "cland-cid"

	// Cookie配置
	CookieMaxAgeOneYear = 31536000 // 1年
	CookiePathRoot      = "/"
	CookieSecure        = true
	CookieHttpOnly      = true

	// 错误码
	ErrorCodeUserInitFailed = 50010010001
)

const (
	SuccessCode         = 200
	ParamErrorBase      = 40000000000
	ParamErrorUserID    = ParamErrorBase + 10010001 // 40010010001
	SystemErrorBase     = 50000000000
	SystemErrorDatabase = SystemErrorBase + 20030002 // 50020030002
)
