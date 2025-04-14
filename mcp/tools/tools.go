package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"time"

	managerhttp "mcp-server-cloudbrowser-go/client/http"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/dreamsxin/mcp-go/mcp"
	"github.com/dreamsxin/mcp-go/server"
)

type ContextKey string

var (
	SessionIDKey = ContextKey("sessionId")
	ApiKeyKey    = ContextKey("apiKey")
)

func SetValueToContext(ctx context.Context, key ContextKey, value string) context.Context {
	return context.WithValue(ctx, key, value)
}

func GetValueFromContext(ctx context.Context, key ContextKey) (string, error) {
	v, ok := ctx.Value(key).(string)
	if !ok {
		return "", fmt.Errorf("missing value for context key %s", key)
	}
	return v, nil
}

// CDPArgs 定义工具调用时的参数
type CDPArgs struct {
	ApiKey    string `json:"apiKey"`
	SessionID string `json:"sessionId"`
	URL       string `json:"url"`
	Name      string `json:"name"`
	Selector  string `json:"selector"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Value     string `json:"value"`
	Script    string `json:"script"`
}

// BrowserSession 存储浏览器会话信息
type BrowserSession struct {
	ctx         context.Context
	allocCancel context.CancelFunc
	cancel      context.CancelFunc
	apiKey      string
}

// 全局会话存储
var Browsers = make(map[string]*BrowserSession)

// createNewBrowserSession 创建新的浏览器会话
func createNewBrowserSession(apiKey, sessionID string) (*BrowserSession, error) {

	client := managerhttp.NewClient()
	browserUrl, err := client.Start(sessionID, apiKey)
	if err != nil {
		log.Println("Failed to create session:", err)
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	return connectBrowserSession(apiKey, sessionID, browserUrl)
}

func connectBrowserSession(apiKey, sessionID, browserUrl string) (*BrowserSession, error) {

	log.Println("-------------------Browser URL:", browserUrl)

	allocCtx, allocCancel := chromedp.NewRemoteAllocator(context.Background(), browserUrl, chromedp.NoModifyURL)
	ctx, cancel := chromedp.NewContext(
		allocCtx,
		chromedp.WithLogf(log.Printf),
	)

	// 启动浏览器
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		allocCancel()
		log.Println("Failed to start browser:", err)
		return nil, fmt.Errorf("failed to start browser: %v", err)
	}

	session := &BrowserSession{
		ctx:         ctx,
		allocCancel: allocCancel,
		cancel:      cancel,
		apiKey:      apiKey,
	}

	Browsers[sessionID] = session
	return session, nil
}

func GetSession(sessionId, apiKey string, autoCreate bool) (*BrowserSession, error) {

	session, exists := Browsers[sessionId]
	if !exists {
		if autoCreate {
			return createNewBrowserSession(apiKey, sessionId)
		}
		return nil, errors.New("session not found")
	}
	if session.apiKey != apiKey {
		return nil, errors.New("invalid apiKey")
	}

	// 判断 session.ctx 是否取消
	select {
	case <-session.ctx.Done():
		delete(Browsers, sessionId)
		session.cancel()      // 关闭浏览器
		session.allocCancel() // 关闭浏览器
		if autoCreate {
			return createNewBrowserSession(apiKey, sessionId)
		}
		return nil, errors.New("session canceled, reconnecting...")
	default:
	}
	return session, nil
}

// 获取当前 URL（无需执行 JavaScript）
func getCurrentURL(session *BrowserSession) (string, error) {
	var url string
	err := chromedp.Run(session.ctx,
		chromedp.Location(&url), // 直接获取地址栏 URL
	)
	return url, err
}

func getPageHTML(session *BrowserSession) (string, error) {
	var htmlContent string
	err := chromedp.Run(session.ctx,
		chromedp.OuterHTML("html", &htmlContent), // 获取整个 HTML 文档
	)
	return htmlContent, err
}

func getPageText(session *BrowserSession) (string, error) {
	var textContent string
	err := chromedp.Run(session.ctx,
		chromedp.Text("html", &textContent), // 获取整个 HTML 文档
	)
	return textContent, err
}

func getElementHtml(session *BrowserSession, selector string) (string, error) {
	var content string
	err := chromedp.Run(session.ctx,
		chromedp.WaitReady(selector), // 等待元素加载完成
		chromedp.OuterHTML(selector, &content, chromedp.ByQuery),
	)
	return content, err
}

func getElementContent(session *BrowserSession, selector string) (string, error) {
	var content string
	err := chromedp.Run(session.ctx,
		chromedp.WaitReady(selector), // 等待元素加载完成
		chromedp.Text(selector, &content, chromedp.ByQuery),
	)
	return content, err
}

func GetSessionId(ctx context.Context, request mcp.CallToolRequest) (string, error) {

	sessionId, err := GetValueFromContext(ctx, SessionIDKey)
	if err == nil {
		return sessionId, nil
	}
	return "", errors.New("invalid sessionId")
}

func GetApiKey(ctx context.Context, request mcp.CallToolRequest) (string, error) {

	apiKey, err := GetValueFromContext(ctx, ApiKeyKey)
	if err == nil {
		return apiKey, nil
	}
	return "", errors.New("Invalid apiKey")
}

func GetUrl(request mcp.CallToolRequest) (string, error) {

	urlstr, ok := request.Params.Arguments["url"].(string)
	if !ok {
		return "", errors.New("Invalid url")
	}
	// 检测 url 格式是否正确
	if _, err := url.Parse(urlstr); err != nil {
		return "", errors.New("Invalid url")
	}
	return urlstr, nil
}

func GetSelector(request mcp.CallToolRequest) (string, error) {

	selector, ok := request.Params.Arguments["selector"].(string)
	if !ok {
		return "", errors.New("Invalid selector")
	}
	return selector, nil
}

func GetValue(request mcp.CallToolRequest) (string, error) {

	value, ok := request.Params.Arguments["value"].(string)
	if !ok {
		return "", errors.New("Invalid value")
	}
	return value, nil
}

func GetScript(request mcp.CallToolRequest) (string, error) {

	script, ok := request.Params.Arguments["script"].(string)
	if !ok {
		return "", errors.New("Invalid script")
	}
	return script, nil
}

// GetValueFromRequest 从请求中获取指定键的值
func GetValueFromRequest(request mcp.CallToolRequest, key string) (string, error) {
	// 尝试从请求参数中获取指定键的值
	value, ok := request.Params.Arguments[key].(string)
	if !ok {
		return "", fmt.Errorf("invalid value for key %s in request", key)
	}
	return value, nil
}

var defaultToolOptions = []mcp.ToolOption{
	// mcp.WithString("sessionId",
	// 	//mcp.Required(),
	// 	mcp.Description("Optional unique identifier for the browser session. Default value will be retrieved from the Authorization header if not provided."),
	// ),
	// mcp.WithString("apiKey",
	// 	//mcp.Required(),
	// 	mcp.Description("Optional authentication key for accessing the cloud browser service. Default value will be retrieved from the Authorization header if not provided."),
	// ),
}

// 注册工具
var Tools = []struct {
	name    string
	params  []mcp.ToolOption
	handler func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error)
}{
	// {
	// 	name: "yunbrowser_start_session",
	// 	params: append([]mcp.ToolOption{
	// 		mcp.WithDescription("Start a cloud browser session"),
	// 	}, defaultToolOptions...),
	// 	handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 		sessionId, err := GetSessionId(ctx, request)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		apiKey, err := GetApiKey(ctx, request)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if session, exists := Browsers[sessionId]; exists {
	// 			if session.apiKey != apiKey {
	// 				return nil, errors.New("invalid apiKey")
	// 			}
	// 			return mcp.NewToolResultText("Session already exists"), nil
	// 		}

	// 		if _, err := createNewBrowserSession(apiKey, sessionId); err != nil {
	// 			return nil, fmt.Errorf("failed to create session: %v", err)
	// 		}
	// 		return mcp.NewToolResultText("Created new browser session"), nil
	// 	},
	// },
	// {
	// 	name: "yunbrowser_stop_session",
	// 	params: append([]mcp.ToolOption{
	// 		mcp.WithDescription("Stop an active browser session"),
	// 	}, defaultToolOptions...),
	// 	handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 		sessionId, err := GetSessionId(ctx, request)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		apiKey, err := GetApiKey(ctx, request)
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if session, exists := Browsers[sessionId]; exists {
	// 			if session.apiKey != apiKey {
	// 				return nil, err
	// 			}
	// 			// client := managerhttp.NewClient(config.Cfg.ServerHttp)
	// 			// err := client.Stop(sessionId, apiKey)
	// 			// if err != nil {
	// 			// 	return nil, fmt.Errorf("failed to close session: %v", err)
	// 			// }
	// 			delete(Browsers, sessionId)
	// 			session.cancel()      // 关闭浏览器
	// 			session.allocCancel() // 关闭浏览器
	// 			return mcp.NewToolResultText("Closed browser session"), nil
	// 		}
	// 		return nil, errors.New("failed to close session: not found")
	// 	},
	// },
	{
		name: "yunbrowser_get_all_tab_urls",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Retrieve the URLs of all open tabs in the browser session"),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			// 获取所有 tab 页的 URL
			var urls []string
			err = chromedp.Run(session.ctx,
				chromedp.Tasks{
					chromedp.ActionFunc(func(ctx context.Context) error {
						// 获取所有 tab 页的目标信息
						targets, err := chromedp.Targets(ctx)
						if err != nil {
							return err
						}
						for _, target := range targets {
							if target.Type == "page" {
								// 获取每个 tab 页的 URL
								targetCtx, _ := chromedp.NewContext(ctx, chromedp.WithTargetID(target.TargetID))
								//defer cancel()
								var url string
								err := chromedp.Run(targetCtx, chromedp.Location(&url))
								if err != nil {
									return err
								}
								urls = append(urls, url)
							}
						}
						return nil
					}),
				},
			)
			if err != nil {
				log.Println("Failed to get all tab URLs:", err)
				return nil, err
			}

			resultJSON, _ := json.MarshalIndent(urls, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", string(resultJSON))), nil
		},
	},
	{
		name: "yunbrowser_get_all_tab_ids",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Retrieve the IDs of all open tabs in the browser session"),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			var tabIds []string
			err = chromedp.Run(session.ctx,
				chromedp.ActionFunc(func(ctx context.Context) error {
					// 获取所有 tab 页的目标信息
					targets, err := chromedp.Targets(ctx)
					if err != nil {
						return err
					}
					for _, target := range targets {
						if target.Type == "page" {
							tabIds = append(tabIds, string(target.TargetID))
						}
					}
					return nil
				}),
			)
			if err != nil {
				log.Println("Failed to get all tab IDs:", err)
				return nil, err
			}

			resultJSON, _ := json.MarshalIndent(tabIds, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", string(resultJSON))), nil
		},
	},
	{
		name: "yunbrowser_switch_to_tab",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Switch to a specified tab in the browser session"),
			mcp.WithString("targetTabId",
				mcp.Required(),
				mcp.Description("ID of the tab to switch to"),
			),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}

			targetTabIdStr, err := GetValueFromRequest(request, "targetTabId")
			if err != nil {
				return nil, err
			}
			targetTabId := target.ID(targetTabIdStr)

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			// 切换到指定 tab
			targetCtx, cancel := chromedp.NewContext(session.ctx, chromedp.WithTargetID(targetTabId))
			session.ctx = targetCtx
			session.cancel = cancel

			// 这里可以添加一些操作来确认切换成功，比如获取当前页面的 URL
			var currentURL string
			err = chromedp.Run(targetCtx, chromedp.Location(&currentURL))
			if err != nil {
				log.Println("Failed to switch to tab:", err)
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf("Switched to tab with ID: %s, current URL: %s", targetTabIdStr, currentURL)), nil
		},
	},
	{
		name: "yunbrowser_navigate",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Navigate the browser to a specified URL"),
			mcp.WithString("url",
				mcp.Required(),
				mcp.Description("URL to navigate to"),
			),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}
			urlstr, err := GetUrl(request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			err = chromedp.Run(session.ctx,
				chromedp.Navigate(urlstr),
				chromedp.Sleep(2*time.Second), // 等待页面加载
			)
			if err != nil {
				log.Println("Failed to navigate:", err)
				return nil, err
			}
			return mcp.NewToolResultText(fmt.Sprintf("Navigated to %s", urlstr)), nil
		},
	},
	{
		name: "yunbrowser_screenshot",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Capture a full-page screenshot of the current browser session"),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			var buf []byte
			err = chromedp.Run(session.ctx,
				chromedp.FullScreenshot(&buf, 70),
			)
			if err != nil {
				log.Println("Failed to take screenshot:", err)
				return nil, errors.New("screenshot failed")
			}

			imageData := base64.StdEncoding.EncodeToString(buf)
			return mcp.NewToolResultImage("Screenshot taken", imageData, "image/png"), nil
		},
	},
	{
		name: "yunbrowser_click",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Perform a click action on a specified DOM element"),
			mcp.WithString("selector",
				mcp.Required(),
				mcp.Description("CSS selector of the element to click"),
			),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}
			selector, err := GetSelector(request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			var nodes []*cdp.Node
			err = chromedp.Run(session.ctx,
				chromedp.Nodes(selector, &nodes, chromedp.ByQuery),
				chromedp.MouseClickNode(nodes[0]),
			)
			if err != nil {
				log.Println("Failed to click:", err)
				return nil, err
			}
			return mcp.NewToolResultText(fmt.Sprintf("Clicked: %s", selector)), nil
		},
	},
	{
		name: "yunbrowser_fill",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Fill a specified input field with a given value"),
			mcp.WithString("selector",
				mcp.Required(),
				mcp.Description("CSS selector of the element to click"),
			),
			mcp.WithString("value",
				mcp.Required(),
				mcp.Description("Value to fill"),
			),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}
			selector, err := GetSelector(request)
			if err != nil {
				return nil, err
			}
			value, err := GetValue(request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			err = chromedp.Run(session.ctx,
				chromedp.SendKeys(selector, value, chromedp.ByQuery),
			)
			if err != nil {
				log.Println("Failed to fill:", err)
				return nil, err
			}
			return mcp.NewToolResultText(fmt.Sprintf("Filled %s with: %s", selector, value)), nil
		},
	},
	{
		name: "yunbrowser_evaluate",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Execute a JavaScript script in the context of the current browser session"),
			mcp.WithString("script",
				mcp.Required(),
				mcp.Description("JavaScript to execute"),
			),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}
			script, err := GetScript(request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			var res *runtime.RemoteObject
			err = chromedp.Run(session.ctx,
				chromedp.Evaluate(script, &res),
			)
			if err != nil {
				log.Println("Failed to evaluate:", err)
				return nil, err
			}

			resultJSON, _ := json.MarshalIndent(res, "", "  ")
			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", string(resultJSON))), nil
		},
	},
	{
		name: "yunbrowser_get_html",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Retrieve the full HTML content of the current page."),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			html, err := getPageHTML(session)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", html)), nil
		},
	},
	{
		name: "yunbrowser_get_content",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Extract all text content from the current page."),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			html, err := getPageText(session)
			if err != nil {
				return nil, err
			}
			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", html)), nil
		},
	},
	{
		name: "yunbrowser_get_element_html",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Retrieve the HTML content of a specified DOM element"),
			mcp.WithString("selector",
				mcp.Required(),
				mcp.Description("CSS selector of the element to get content from"),
			),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}
			selector, err := GetSelector(request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			content, err := getElementHtml(session, selector)
			if err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", content)), nil
		},
	},
	{
		name: "yunbrowser_get_element_content",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Retrieve the text content of a specified DOM element"),
			mcp.WithString("selector",
				mcp.Required(),
				mcp.Description("CSS selector of the element to get content from"),
			),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}
			selector, err := GetSelector(request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			content, err := getElementContent(session, selector)
			if err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", content)), nil
		},
	},
	{
		name: "yunbrowser_get_current_url",
		params: append([]mcp.ToolOption{
			mcp.WithDescription("Retrieve the current URL of the browser page"),
		}, defaultToolOptions...),
		handler: func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionId, err := GetSessionId(ctx, request)
			if err != nil {
				return nil, err
			}
			apiKey, err := GetApiKey(ctx, request)
			if err != nil {
				return nil, err
			}

			session, err := GetSession(sessionId, apiKey, true)
			if err != nil {
				return nil, err
			}

			currentURL, err := getCurrentURL(session)
			if err != nil {
				return nil, err
			}

			return mcp.NewToolResultText(fmt.Sprintf("Execution result:\n%s", currentURL)), nil
		},
	},
}

func InitTools(server *server.MCPServer) {
	for _, t := range Tools {
		tool := mcp.NewTool(t.name,
			t.params...,
		)
		server.AddTool(tool, t.handler)
	}
}
