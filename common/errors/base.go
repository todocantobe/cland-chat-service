package errors

type Error struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

var Err400 = Error{Code: 40010010000, Msg: "Invalid parameter"}
var ErrUserIDMissing = Error{Code: 40010010001, Msg: "Invalid parameter: user_id is missing"}

var Err500 = Error{Code: 50010010000, Msg: "系统异常"}
