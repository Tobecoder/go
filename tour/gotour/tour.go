package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Go-zh/tools/godoc/static"
	"github.com/Go-zh/tools/present"
)

// 定义变量
var (
	UiContent []byte
)

// initTour 初始化课程相关信息，主要是渲染模板
func initTour(root, transport string) error {
	// 渲染前保证playground可用
	present.PlayEnabled = true

	// 安装模版
	//	action := filepath.Join(root, "template", "action.tmpl")
	//	tmpl, err := template.ParseFiles(action)
	//	if err != nil {
	//		return fmt.Errorf("parse %v", err)
	//	}

	// 初始化UI
	index := filepath.Join(root, "template", "index.tmpl")
	indexTmpl, err := template.ParseFiles(index)
	if err != nil {
		return fmt.Errorf("parse templates: %v", err)
	}
	buf := new(bytes.Buffer)

	data := struct {
		Transport  template.JS
		SocketAddr string
	}{template.JS(transport), socketAddr()}

	if err = indexTmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("render UI: %v", err)
	}
	//http请求策略
	httpSchedule()

	UiContent = buf.Bytes()
	return initScript(root)
}

func httpSchedule() {

}

// initScript 初始化前端脚本
func initScript(root string) error {
	// 初始化buffer
	buf := new(bytes.Buffer)

	content, ok := static.Files["playground.js"]
	if !ok {
		return fmt.Errorf("playground.js not found in static files")
	}
	buf.WriteString(content)

	// 注意脚本顺序
	js := []string{
		"static/lib/jquery.min.js",
		"static/lib/jquery-ui.min.js",
		"static/lib/angular.min.js",
		"static/lib/codemirror/lib/codemirror.js",
		"static/lib/codemirror/mode/go/go.js",
		"static/lib/angular-ui.min.js",
		"static/js/app.js",
		"static/js/controllers.js",
		"static/js/directives.js",
		"static/js/services.js",
		"static/js/values.js",
	}
	for _, file := range js {
		script, err := ioutil.ReadFile(filepath.Join(root, file))
		if err != nil {
			return fmt.Errorf("couldn't open %v: %v", file, err)
		}
		_, err = buf.Write(script)
		if err != nil {
			return fmt.Errorf("error concatenating %v: %v", file, err)
		}
	}

	var gzBuf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&gzBuf, gzip.BestCompression)
	if err != nil {
		return err
	}
	gz.Write(buf.Bytes())
	gz.Close()

	http.HandleFunc("/script.js", func(w http.ResponseWriter, r *http.Request) {
		modTime := time.Now()
		// 设置返回头信息
		w.Header().Set("Content-type", "application/javascript")
		w.Header().Set("Cache-control", "max-age=604800")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			http.ServeContent(w, r, "", modTime, bytes.NewReader(gzBuf.Bytes()))
		} else {
			http.ServeContent(w, r, "", modTime, bytes.NewReader(buf.Bytes()))
		}
	})
	return nil
}

// renderUI 渲染UI内容到终端
func renderUI(w io.Writer) error {
	if UiContent == nil {
		panic("renderUI called before successful initTour")
	}
	_, err := w.Write(UiContent)
	return err
}

// socketAddr 获取socket服务地址
func socketAddr() string {
	return "ws://" + httpAddr + SocketPath
}

// waitServer 服务器是否准备就绪
func waitServer(url string) bool {
	times := 20
	for times > 0 {
		resp, err := http.Get(url)
		if err == nil {
			resp.Body.Close()
			return true
		}
		times--
	}
	return false
}

func startBrowser(url string) bool {
	// try to start the browser
	var args []string
	switch runtime.GOOS {
	case "darwin":
		args = []string{"open"}
	case "windows":
		args = []string{"cmd", "/c", "start"}
	default:
		args = []string{"xdg-open"}
	}
	cmd := exec.Command(args[0], append(args[1:], url)...)
	return cmd.Start() == nil
}
