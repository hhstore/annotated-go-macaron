// +build go1.3

// Copyright 2014 The Macaron Authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

// Package macaron is a high productive and modular web framework in Go.
package macaron

import (
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/Unknwon/com"
	"gopkg.in/ini.v1"

	"github.com/go-macaron/inject"
)

const _VERSION = "1.1.8.0826"

func Version() string {
	return _VERSION
}

//-----------------------------------------------------
// Handler 类型: 空接口
// 	- 支持传入任意类型, 但要求传入类型为: 可调用函数
// Handler can be any callable function.
// Macaron attempts to inject services into the handler's argument list,
// and panics if an argument could not be fullfilled via dependency injection.
type Handler interface{}

// 校验 Handler 类型.
// validateHandler makes sure a handler is a callable function,
// and panics if it is not.
func validateHandler(h Handler) {
	if reflect.TypeOf(h).Kind() != reflect.Func {
		panic("Macaron handler must be a callable function")
	}
}

// 批量校验
// validateHandlers makes sure handlers are callable functions,
// and panics if any of them is not.
func validateHandlers(handlers []Handler) {
	for _, h := range handlers {
		validateHandler(h)
	}
}

//-----------------------------------------------------


// Macaron represents the top level web application.
// inject.Injector methods can be invoked to map services on a global level.
type Macaron struct {
	inject.Injector              // 反射
	befores      []BeforeHandler // Handler 钩子
	handlers     []Handler       // Handler 集合
	action       Handler

	hasURLPrefix bool            // 是否有 URL 前缀
	urlPrefix    string          // For suburl support.
	*Router                      // 路由模块, 注意Router的定义, (类似链表结构)
				     // 这里需要理解一下: 是 嵌入, 不是 循环引用, 跟Macaron定义不冲突

	logger       *log.Logger     // 日志记录器
}

// NewWithLogger creates a bare bones Macaron instance.
// Use this method if you want to have full control over the middleware that is used.
// You can specify logger output writer with this function.
func NewWithLogger(out io.Writer) *Macaron {
	m := &Macaron{
		Injector: inject.New(),
		action:   func() {},
		Router:   NewRouter(),
		logger:   log.New(out, "[Macaron] ", 0),
	}
	m.Router.m = m
	m.Map(m.logger)
	m.Map(defaultReturnHandler())
	m.NotFound(http.NotFound)
	m.InternalServerError(func(rw http.ResponseWriter, err error) {
		http.Error(rw, err.Error(), 500)
	})
	return m
}

// New creates a bare bones Macaron instance.
// Use this method if you want to have full control over the middleware that is used.
func New() *Macaron {
	return NewWithLogger(os.Stdout)
}

// Classic creates a classic Macaron with some basic default middleware:
// mocaron.Logger, mocaron.Recovery and mocaron.Static.
func Classic() *Macaron {
	m := New()
	m.Use(Logger())         // 添加到 handler 列表
	m.Use(Recovery())       // 添加到 handler 列表
	m.Use(Static("public")) // 添加到 handler 列表
	return m
}

// Handlers sets the entire middleware stack with the given Handlers.
// This will clear any current middleware handlers,
// and panics if any of the handlers is not a callable function
func (m *Macaron) Handlers(handlers ...Handler) {
	m.handlers = make([]Handler, 0)
	for _, handler := range handlers {
		m.Use(handler) // 添加到 handler 列表
	}
}

// Action sets the handler that will be called after all the middleware has been invoked.
// This is set to macaron.Router in a macaron.Classic().
func (m *Macaron) Action(handler Handler) {
	validateHandler(handler)  // 校验
	m.action = handler  // 激活
}

//*************************************
// 钩子函数类型定义
// 在 handler 调用之前执行
// BeforeHandler represents a handler executes at beginning of every request.
// Macaron stops future process when it returns true.
type BeforeHandler func(rw http.ResponseWriter, req *http.Request) bool

// 添加钩子集合:
// 将所有 `BeforeHandler` 类型的函数钩子, 添加到 befores 列表.
func (m *Macaron) Before(handler BeforeHandler) {
	m.befores = append(m.befores, handler)
}


// 关键方法:
// Use adds a middleware Handler to the stack,
// and panics if the handler is not a callable func.
// Middleware Handlers are invoked in the order that they are added.
func (m *Macaron) Use(handler Handler) {
	validateHandler(handler) // 校验
	m.handlers = append(m.handlers, handler)  // 添加到方法列表
}


// 请求上下文创建:
//	- 类似 flask 的上下文概念
// 	- 这部分代码的实现, 注意阅读, 不好理解
func (m *Macaron) createContext(rw http.ResponseWriter, req *http.Request) *Context {
	c := &Context{
		Injector: inject.New(),
		handlers: m.handlers,
		action:   m.action,
		index:    0,
		Router:   m.Router,
		Req:      Request{req},
		Resp:     NewResponseWriter(rw),
		Render:   &DummyRender{rw},
		Data:     make(map[string]interface{}),
	}
	c.SetParent(m)		// 关键方法
	c.Map(c)
	c.MapTo(c.Resp, (*http.ResponseWriter)(nil))
	c.Map(req)
	return c
}

//*************************************
// 关键方法:
// ServeHTTP is the HTTP Entry point for a Macaron instance.
// Useful if you want to control your own HTTP server.
// Be aware that none of middleware will run without registering any router.
func (m *Macaron) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if m.hasURLPrefix {
		req.URL.Path = strings.TrimPrefix(req.URL.Path, m.urlPrefix)
	}

	// handler 钩子方法列表不空, 先行执行钩子方法
	for _, h := range m.befores {
		if h(rw, req) {
			return
		}
	}

	// 启动服务
	m.Router.ServeHTTP(rw, req)
}

func GetDefaultListenInfo() (string, int) {
	host := os.Getenv("HOST")
	if len(host) == 0 {
		host = "0.0.0.0"	// 默认 IP
	}
	port := com.StrTo(os.Getenv("PORT")).MustInt()
	if port == 0 {
		port = 4000	// 默认端口
	}
	return host, port
}

//*************************************
// 模块入口:
//
// Run the http server. Listening on os.GetEnv("PORT") or 4000 by default.
func (m *Macaron) Run(args ...interface{}) {
	host, port := GetDefaultListenInfo()	// 获取默认IP+端口
	if len(args) == 1 {
		switch arg := args[0].(type) {
		case string:
			host = arg
		case int:
			port = arg
		}
	} else if len(args) >= 2 {
		if arg, ok := args[0].(string); ok {
			host = arg
		}
		if arg, ok := args[1].(int); ok {
			port = arg
		}
	}

	addr := host + ":" + com.ToStr(port)	// IP + 端口
	logger := m.GetVal(reflect.TypeOf(m.logger)).Interface().(*log.Logger)
	logger.Printf("listening on %s (%s)\n", addr, safeEnv())
	logger.Fatalln(http.ListenAndServe(addr, m))	// 启动监听服务
}

// SetURLPrefix sets URL prefix of router layer, so that it support suburl.
func (m *Macaron) SetURLPrefix(prefix string) {
	m.urlPrefix = prefix
	m.hasURLPrefix = len(m.urlPrefix) > 0
}

// ____   ____            .__      ___.   .__
// \   \ /   /____ _______|__|____ \_ |__ |  |   ____   ______
//  \   Y   /\__  \\_  __ \  \__  \ | __ \|  | _/ __ \ /  ___/
//   \     /  / __ \|  | \/  |/ __ \| \_\ \  |_\  ___/ \___ \
//    \___/  (____  /__|  |__(____  /___  /____/\___  >____  >
//                \/              \/    \/          \/     \/

const (
	DEV = "development"
	PROD = "production"
	TEST = "test"
)

var (
	// Env is the environment that Macaron is executing in.
	// The MACARON_ENV is read on initialization to set this variable.
	Env = DEV
	envLock sync.Mutex

	// Path of work directory.
	Root string

	// Flash applies to current request.
	FlashNow bool

	// Configuration convention object.
	cfg *ini.File
)

func setENV(e string) {
	envLock.Lock()
	defer envLock.Unlock()

	if len(e) > 0 {
		Env = e
	}
}

func safeEnv() string {
	envLock.Lock()
	defer envLock.Unlock()

	return Env
}

func init() {
	setENV(os.Getenv("MACARON_ENV"))

	var err error
	Root, err = os.Getwd()
	if err != nil {
		panic("error getting work directory: " + err.Error())
	}
}

// SetConfig sets data sources for configuration.
func SetConfig(source interface{}, others ...interface{}) (_ *ini.File, err error) {
	cfg, err = ini.Load(source, others...)
	return Config(), err
}

// Config returns configuration convention object.
// It returns an empty object if there is no one available.
func Config() *ini.File {
	if cfg == nil {
		return ini.Empty()
	}
	return cfg
}
