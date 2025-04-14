package http

import (
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	httpclient *resty.Client
}

func NewClient() *Client {

	httpclient := resty.New().SetTimeout(60 * time.Second)
	return &Client{
		httpclient: httpclient,
	}
}

func (c *Client) BuildURL(path string) string {

	u, err := url.JoinPath("https://cloud.yunlogin.com/v2/cloudbrowser", path)
	if err != nil {
		panic(err)
	}

	return u
}

func (c *Client) Start(sessionId, apiKey string) (browserUrl string, err error) {

	res, err := c.httpclient.R().
		SetQueryString(fmt.Sprintf("sessionId=%s&apiKey=%s", sessionId, apiKey)).
		SetResult(&Resp{}).
		Post(c.BuildURL("api/session/start"))

	if err != nil {
		log.Println(err)
		return "", err
	}
	log.Printf("%v", res)
	if res.IsError() {
		return "", fmt.Errorf("start session error: %s", res.Body())
	}
	ret := res.Result().(*Resp)
	if ret == nil {
		return "", fmt.Errorf("start session error: %s", res.Body())
	}

	v, ok := ret.Data.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("start session error: %s", res.Body())
	}
	browserUrl, _ = v["browserUrl"].(string)
	return
}
