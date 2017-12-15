package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Go-zh/tools/playground/socket"
)

// 定义基本常量
const (
	RootPath   = "github.com/Tobecoder/go/tour"
	SocketPath = "/socket"
)

const localhostWarning = `
WARNING!  WARNING!  WARNING!

I appear to be listening on an address that is not localhost.
Anyone with access to this address and port will have access
to this machine as the user running gotour.

If you don't understand this message, hit Control-C to terminate this process.

WARNING!  WARNING!  WARNING!
`

// 定义全局变量
var (
	httpListen  = flag.String("http", "127.0.0.1:3999", "ip and port")
	openBrowser = flag.Bool("open", true, "open the default browser")
)

var (
	// GOPATH containing the tour packages
	gopath = os.Getenv("GOPATH")

	httpAddr string
)

// isRoot 检测路径path是否为根目录，依据标准为是否包含欢迎页
func isRoot(path string) bool {
	_, err := os.Stat(filepath.Join(path, "content", "welcome.article"))
	if err == nil {
		_, err = os.Stat(filepath.Join(path, "template", "index.tmpl"))
	}
	return err == nil
}

// findRoot 查找根目录
func findRoot() (string, error) {
	context := build.Default
	p, err := context.Import(RootPath, "", build.FindOnly)
	if err == nil && isRoot(p.Dir) {
		return p.Dir, nil
	}
	tourRoot := filepath.Join(runtime.GOROOT(), "misc", "tour")
	context.GOROOT = tourRoot
	p, err = context.Import(RootPath, "", build.FindOnly)
	if err == nil && isRoot(tourRoot) {
		gopath = tourRoot
		return tourRoot, nil
	}
	return "", fmt.Errorf("Couldn't find go-tour content; check $GOROOT and $GOPATH")
}

func main() {
	// 解析命令行参数
	flag.Parse()

	// 查找根路径
	root, err := findRoot()
	if err != nil {
		log.Fatalf("Couldn't find tour files: %v", err)
	}

	log.Println("Serving content from ", root)

	// 处理主机和端口
	host, port, err := net.SplitHostPort(*httpListen)

	if err != nil {
		log.Fatalln(err)
	}

	if host != "127.0.0.1" && host != "localhost" {
		log.Print(localhostWarning)
	}

	// 设置http地址
	httpAddr = host + ":" + port

	// 初始化
	if err := initTour(root, "SocketTransport"); err != nil {
		log.Fatal(err)
	}
	// 解析url根目录
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if err := renderUI(w); err != nil {
			log.Println(err)
		}
	})

	// 监听静态文件
	static := http.FileServer(http.Dir(root))
	http.Handle("/static/", static)

	//监听socket
	origin := &url.URL{Scheme: "http", Host: httpAddr}
	http.Handle(SocketPath, socket.NewHandler(origin))

	// 启动浏览器
	go func() {
		url := "http://" + httpAddr
		if waitServer(url) && *openBrowser && startBrowser(url) {
			log.Printf("A browser window should open. If not, please visit %s", url)
		} else {
			log.Printf("Please open your web browser and visit %s", url)
		}
	}()
	// 监听服务
	log.Fatal(http.ListenAndServe(httpAddr, nil))
}

func init() {
	socket.Environ = environ
}

// environ returns the original execution environment with GOPATH
// replaced (or added) with the value of the global var gopath.
func environ() (env []string) {
	for _, v := range os.Environ() {
		if !strings.HasPrefix(v, "GOPATH=") {
			env = append(env, v)
		}
	}
	env = append(env, "GOPATH="+gopath)
	return
}
