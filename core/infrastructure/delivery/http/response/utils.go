package response

func Success(data interface{}) Response {
	return Response{Code: 200, Msg: "Success", Data: data}
}
func Error(code int, msg string, data interface{}) Response {
	return Response{Code: code, Msg: msg, Data: data}
}
