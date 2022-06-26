/*
@Time       : 2022/1/7
@Author     : wuqiusheng
@File       : http_server.go
@Description: 启动http服务 get post
*/
package easynet

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
)

func GetHttpClientIp(r *http.Request) string {
	clientIP := r.Header.Get("X-Forwarded-For")
	clientIP = strings.TrimSpace(strings.Split(clientIP, ",")[0])
	if clientIP == "" {
		clientIP = strings.TrimSpace(r.Header.Get("X-Real-Ip"))
	}
	if clientIP != "" {
		return clientIP
	}
	//if addr :=r.Header.Get("X-Appengine-Remote-Addr"); addr != "" {
	//	return addr
	//}
	if ip, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr)); err == nil {
		return ip
	}
	return ""
}

//参数解析 GET or (POST PUT PATCH application/x-www-form-urlencoded)
func ParseForm(r *http.Request) {
	r.ParseForm()
}

//参数解析 POST request body as multipart/form-data.
func ParseMultipartForm(r *http.Request) {
	r.ParseMultipartForm(10000)
}

//读取参数
func GetHttpParam(r *http.Request, key string) string {
	value, _ := url.QueryUnescape(r.Form.Get(key))
	return value
}

//读取参数 json格式
func GetHttpBody(r *http.Request) ([]byte, error) {
	return ioutil.ReadAll(r.Body)
}

//发送返回值
func SendHttpResponseWithStatus(status int, w http.ResponseWriter, m map[string]interface{}) {
	if err, ok := m["error"]; ok {
		if err, ok := err.(*Error); ok {
			m["error"] = GetErrId(err)
		}
	}
	data, _ := json.MarshalIndent(m, "", "\t") //easyutil.JsonPack(m)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(status)
	w.Write(data)
}

//发送返回值
func SendHttpJsonResponse(w http.ResponseWriter, m interface{}) {
	data, _ := json.MarshalIndent(m, "", "\t") //easyutil.JsonPack(m)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

//发送返回值
func SendHttpResponse(w http.ResponseWriter, m map[string]interface{}) {
	if err, ok := m["error"]; ok {
		if err, ok := err.(*Error); ok {
			m["error"] = GetErrId(err)
		}
	}
	data, _ := json.MarshalIndent(m, "", "\t") //easyutil.JsonPack(m)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Write(data)
}

func SendHttpError(w http.ResponseWriter, err error) {
	m := map[string]interface{}{
		"error": GetErrId(err),
	}
	SendHttpResponse(w, m)
}

func SendHttpErrorWithInfo(w http.ResponseWriter, err error, info string) {
	m := map[string]interface{}{
		"error": GetErrId(err),
		"info":  info,
	}
	SendHttpResponse(w, m)
}

func SendHttpErrorWithData(w http.ResponseWriter, err error, m map[string]interface{}) {
	m["error"] = GetErrId(err)
	SendHttpResponse(w, m)
}

func tryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		Try(func() {
			next.ServeHTTP(w, r)
		}, func(err interface{}) {
			LogError("http process msg panic err:%v", err)
			if e, ok := err.(*Error); ok {
				m := map[string]interface{}{
					"error": e.Id,
				}
				SendHttpResponse(w, m)
			}
		})
	})
}

//路由表
type RouteMap = map[string]func(w http.ResponseWriter, r *http.Request)

//http启动参数
type HttpStartParam struct {
	Addr       string   //ip:pport
	UseHttps   bool     //是否使用https
	SSLCrtPath string   //SSLCrt路径
	SSLKeyPath string   //SSLKey路径
	RouteMap   RouteMap //路由表
}

//启动http服务
func StartHttp(startParam *HttpStartParam) {
	for k, v := range startParam.RouteMap {
		http.Handle(k, tryHandler(http.HandlerFunc(v)))
	}
	s := &http.Server{Addr: startParam.Addr}
	Go(func() {
		if startParam.UseHttps {
			if startParam.SSLCrtPath == "" || startParam.SSLKeyPath == "" {
				LogWarn("server start https failed auto change to http")
				err := s.ListenAndServe()
				if err != nil {
					LogError("server http start failed err:%v", err)
				}
			} else {
				err := s.ListenAndServeTLS(startParam.SSLCrtPath, startParam.SSLKeyPath)
				if err != nil {
					LogError("server https start failed err:%v", err)
				}
			}
		} else {
			err := s.ListenAndServe()
			if err != nil {
				LogError("server http start failed err:%v", err)
			}
		}
	})

	//统一管理
	Go2(func(cstop chan struct{}) {
		select {
		case <-cstop:
			s.Close()
		}
	})
}

//示例
func httpTest(w http.ResponseWriter, r *http.Request) {
	ParseForm(r)
	test := GetHttpParam(r, "test")
	SendHttpResponse(w, map[string]interface{}{
		"v":     "ok",
		"param": test,
	})
}

// 127.0.0.1:5000/api/v1/test?test=1
func TestHttp() {
	routeMap := map[string]func(w http.ResponseWriter, r *http.Request){
		"/api/v1/test": httpTest,
	}
	startParam := &HttpStartParam{
		Addr:     ":5000",
		RouteMap: routeMap,
	}
	StartHttp(startParam)
}
