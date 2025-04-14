package http

type ErrorRes struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type CreateRes struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data CreateResData `json:"data"`
}

type CreateResData struct {
	Id string `json:"id"`
}

type DeleteRes struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

type StartRes struct {
	Code int          `json:"code"`
	Msg  string       `json:"msg"`
	Data StartResData `json:"data"`
}

type StartResData struct {
	Url string `json:"url"`
}

type StatusRes struct {
	Code int           `json:"code"`
	Msg  string        `json:"msg"`
	Data StatusResData `json:"data"`
}
type StatusResData struct {
	Status int    `json:"status"`
	Url    string `json:"url"`
	Msg    string `json:"msg"`
}

type StopRes struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}
