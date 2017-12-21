package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Go-zh/tools/godoc/static"
	"github.com/Tobecoder/go/tools/present"
	"encoding/json"
	"crypto/sha1"
	"encoding/base64"
)

// 定义变量
var (
	uiContent      []byte
	Lessons        = make(map[string][]byte)
	lessonNotFound = fmt.Errorf("lesson not found")
)

// initTour 初始化课程相关信息，主要是渲染模板
func initTour(root, transport string) error {
	// 渲染前保证playground可用
	present.PlayEnabled = true

	// 安装模版 ---- present.Template()为什么必须用这个
	action := filepath.Join(root, "template", "action.tmpl")
	tmpl, err := present.Template().ParseFiles(action)
	if err != nil {
		return fmt.Errorf("parse %v", err)
	}

	//初始化课程
	contentPath := filepath.Join(root, "content")
	if err = initLessons(tmpl, contentPath); err != nil {
		return fmt.Errorf("init lessons %v", err)
	}

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

	uiContent = buf.Bytes()
	return initScript(root)
}

func initLessons(tmpl *template.Template, content string) error {
	f, err := os.Open(content)
	if err != nil {
		return err
	}
	files, err := f.Readdirnames(0)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !strings.HasSuffix(file, ".article") {
			continue
		}
		article, err := parseLessons(tmpl, filepath.Join(content, file))
		if err != nil {
			return fmt.Errorf("parsing %v: %v", file, err)
		}
		name := strings.TrimSuffix(file, ".article")
		Lessons[name] = article
	}
	return nil
}

type Lesson struct {
	Title       string
	Description string
	Pages       []Page
}

type Page struct {
	Title   string
	Content string
	Files   []File
}

type File struct {
	Name    string
	Content string
	Hash    string
}

func parseLessons(tmpl *template.Template, path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	doc, err := present.Parse(f, path, 0)
	if err != nil {
		return nil, err
	}
	lesson := Lesson{
		doc.Title,
		doc.Subtitle,
		make([]Page, len(doc.Sections)),
	}

	for i, v := range doc.Sections {
		p := &lesson.Pages[i]
		p.Title = v.Title
		codes := findPlayCode(sec)
		p.Files = make
	}

	for i, sec := range doc.Sections {
		p := &lesson.Pages[i]
		w := new(bytes.Buffer)
		if err := sec.Render(w, tmpl); err != nil {
			return nil, fmt.Errorf("render section: %v", err)
		}
		p.Title = sec.Title
		p.Content = w.String()
		codes := findPlayCode(sec)
		p.Files = make([]File, len(codes))
		for i, c := range codes {
			f := &p.Files[i]
			f.Name = c.FileName
			f.Content = string(c.Raw)
			hash := sha1.Sum(c.Raw)
			f.Hash = base64.StdEncoding.EncodeToString(hash[:])
		}
	}

	w := new(bytes.Buffer)
	err = json.NewEncoder(w).Encode(lesson)
	if err != nil {
		return nil, fmt.Errorf("encode lesson: %v", err)
	}
	return w.Bytes(), nil
}

//// initLessons 初始化课程导航
//func initLessons(tmpl *template.Template, content string) error {
//	dir, err := os.Open(content)
//	if err != nil {
//		return err
//	}
//	files, err := dir.Readdirnames(0)
//	if err != nil {
//		return err
//	}

//	for _, f := range files {
//		if filepath.Ext(f) != ".article" {
//			continue
//		}
//		content, err := parseLesson(tmpl, filepath.Join(content, f))
//		if err != nil {
//			return fmt.Errorf("parsing %v: %v", f, err)
//		}
//		name := strings.TrimSuffix(f, ".article")
//		Lessons[name] = content
//	}
//	return nil
//}

//// File defines the JSON form of a code file in a page.
//type File struct {
//	Name    string
//	Content string
//	Hash    string
//}

//// Page defines the JSON form of a tour lesson page.
//type Page struct {
//	Title   string
//	Content string
//	Files   []File
//}

//// Lesson defines the JSON form of a tour lesson.
//type Lesson struct {
//	Title       string
//	Description string
//	Pages       []Page
//}

//// parseLesson parses and returns a lesson content given its name and
//// the template to render it.
//func parseLesson(tmpl *template.Template, path string) ([]byte, error) {
//	file, err := os.Open(path)
//	if err != nil {
//		return nil, err
//	}
//	defer file.Close()
//	doc, err := present.Parse(file, path, 0)
//	if err != nil {
//		return nil, err
//	}

//	lesson := Lesson{
//		doc.Title,
//		doc.Subtitle,
//		make([]Page, len(doc.Sections)),
//	}

//	for i, sec := range doc.Sections {
//		p := &lesson.Pages[i]
//		w := new(bytes.Buffer)
//		if err := sec.Render(w, tmpl); err != nil {
//			return nil, fmt.Errorf("render section: %v", err)
//		}
//		p.Title = sec.Title
//		p.Content = w.String()
//		codes := findPlayCode(sec)
//		p.Files = make([]File, len(codes))
//		for i, c := range codes {
//			f := &p.Files[i]
//			f.Name = c.FileName
//			f.Content = string(c.Raw)
//			hash := sha1.Sum(c.Raw)
//			f.Hash = base64.StdEncoding.EncodeToString(hash[:])
//		}
//	}

//	w := new(bytes.Buffer)
//	if err := json.NewEncoder(w).Encode(lesson); err != nil {
//		return nil, fmt.Errorf("encode lesson: %v", err)
//	}
//	return w.Bytes(), nil
//}

//// findPlayCode returns a slide with all the Code elements in the given
//// Elem with Play set to true.
//func findPlayCode(e present.Elem) []*present.Code {
//	var r []*present.Code
//	switch v := e.(type) {
//	case present.Code:
//		if v.Play {
//			r = append(r, &v)
//		}
//	case present.Section:
//		for _, s := range v.Elem {
//			r = append(r, findPlayCode(s)...)
//		}
//	}
//	return r
//}

// writeLesson writes the tour content to the provided Writer.
// 流程需要详细了解
func writeLesson(name string, w io.Writer) error {
	if uiContent == nil {
		panic("writeLesson called before successful initTour")
	}
	if len(name) == 0 {
		return writeAllLessons(w)
	}
	l, ok := Lessons[name]
	if !ok {
		return lessonNotFound
	}
	_, err := w.Write(l)
	return err
}

func writeAllLessons(w io.Writer) error {
	if _, err := fmt.Fprint(w, "{"); err != nil {
		return err
	}
	length := len(Lessons)
	for k, v := range Lessons {
		if _, err := fmt.Fprintf(w, "%q:%s", k, v); err != nil {
			return err
		}
		length--
		if length > 0 {
			if _, err := fmt.Fprint(w, ","); err != nil {
				return err
			}
		}
	}
	_, err := fmt.Fprint(w, "}")
	return err
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
	if uiContent == nil {
		panic("renderUI called before successful initTour")
	}
	_, err := w.Write(uiContent)
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
		time.Sleep(100 * time.Millisecond)
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
