package http

type PingReq struct {
	CloudId string `json:"cloudId"`
}

type StartReq struct {
	CloudId      string `json:"cloudId"`
	SessionId    string `json:"sessionId" binding:"required"`
	SessionLogId uint64 `json:"sessionLogId"`
	ApiKey       string `json:"apiKey"`
}

type Resp struct {
	ReqId string `json:"reqId,omitempty"` //`json:"请求id"`
	Code  int    `json:"code"`            //返回码
	Msg   string `json:"msg,omitempty"`   //消息
	Data  any    `json:"data,omitempty"`  //数据
}
