package errors

type Error struct {
	Code int
	Msg  string
}

var ErrUserIDMissing = Error{Code: 40010010001, Msg: "Invalid parameter: user_id is missing"}
