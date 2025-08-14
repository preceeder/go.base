package forkProcess

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

var (
	listener net.Listener
	server   *http.Server
	connWg   sync.WaitGroup // 跟踪活跃连接
)

// 自定义连接包装器，用来统计连接数
type trackedConn struct {
	net.Conn
}

func (c *trackedConn) Close() error {
	defer connWg.Done()
	return c.Conn.Close()
}

// 连接跟踪的 Listener
type trackedListener struct {
	net.Listener
}

func (tl *trackedListener) Accept() (net.Conn, error) {
	conn, err := tl.Listener.Accept()
	if err != nil {
		return nil, err
	}
	connWg.Add(1)
	return &trackedConn{Conn: conn}, nil
}

func main() {
	isChild := os.Getenv("GRACEFUL_RESTART") == "true"

	var err error
	if isChild {
		fmt.Println("[新进程] 从 FD 继承监听")
		f := os.NewFile(uintptr(3), "")
		listener, err = net.FileListener(f)
		if err != nil {
			panic(err)
		}
	} else {
		fmt.Println("[旧进程] 正常监听端口")
		listener, err = net.Listen("tcp", ":8080")
		if err != nil {
			panic(err)
		}
	}

	// 包装成可跟踪连接的 listener
	tl := &trackedListener{Listener: listener}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "PID: %d, Time: %s\n", os.Getpid(), time.Now().Format(time.RFC3339))
		time.Sleep(3 * time.Second) // 模拟长连接
	})

	mux.HandleFunc("/restart", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "平滑重启中...")

		execPath, _ := os.Executable()
		tcpListener := listener.(*net.TCPListener)
		file, _ := tcpListener.File()

		cmd := exec.Command(execPath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.ExtraFiles = []*os.File{file}
		cmd.Env = append(os.Environ(), "GRACEFUL_RESTART=true")

		if err := cmd.Start(); err != nil {
			fmt.Println("启动新进程失败:", err)
			return
		}
		fmt.Println("[旧进程] 新进程 PID:", cmd.Process.Pid)

		// 关闭监听，不接收新连接
		listener.Close()

		// 等待所有连接完成或超时
		done := make(chan struct{})
		go func() {
			connWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			fmt.Println("[旧进程] 所有连接处理完成，退出")
		case <-time.After(10 * time.Second):
			fmt.Println("[旧进程] 超时退出，可能有未完成连接")
		}

		os.Exit(0)
	})

	server = &http.Server{
		Handler: mux,
	}

	fmt.Printf("[PID %d] 服务启动\n", os.Getpid())
	if err := server.Serve(tl); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}
