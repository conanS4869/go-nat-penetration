package main

import (
	"github.com/gin-gonic/gin"
	"go-nat-penetration/define"
	"go-nat-penetration/helper"
	"go-nat-penetration/service"
	"io"
	"log"
	"net"
	"sync"
)

var controlConn *net.TCPConn
var userConn *net.TCPConn
var wg sync.WaitGroup

// 将server打包,放置在外网服务器并运行
func main() {
	wg.Add(1)
	// 控制中心监听
	go controlListen()
	// 用户请求的监听
	go userRequestListen()
	// 隧道监听
	go tunnelListen()
	// 启动Web服务
	go runGin()
	wg.Wait()
}

func controlListen() {
	tcpListener, err := helper.CreateListen(define.ControlServerAddr)
	if err != nil {
		panic(err)
	}
	log.Printf("[控制中心] 监听中：%v\n", tcpListener.Addr().String())
	for {
		controlConn, err = tcpListener.AcceptTCP()
		if err != nil {
			log.Printf("ControlListen AcceptTCP Error:%v\n", err)
			return
		}
		go helper.KeepAlive(controlConn)
	}
}

func userRequestListen() {
	tcpListener, err := helper.CreateListen(define.UserRequestAddr)
	if err != nil {
		panic(err)
	}
	log.Printf("[用户请求] 监听中：%v\n", tcpListener.Addr().String())
	for {
		userConn, err = tcpListener.AcceptTCP()
		if err != nil {
			log.Printf("UserRequestListen AcceptTCP Error:%v\n", err)
			return
		}
		// 发送消息，告诉客户端有新的连接
		_, err := controlConn.Write([]byte(define.NewConnection))
		if err != nil {
			log.Printf("发送失败: %v", err)
		}
	}
}

func tunnelListen() {
	tcpListener, err := helper.CreateListen(define.TunnelServerAddr)
	if err != nil {
		panic(err)
	}
	log.Printf("[隧道] 监听中：%v\n", tcpListener.Addr().String())
	for {
		conn, err := tcpListener.AcceptTCP()
		if err != nil {
			log.Printf("unnelListen AcceptTCP Error:%v\n", err)
			return
		}
		// 数据转发
		go io.Copy(userConn, conn)
		go io.Copy(conn, userConn)
	}
}

func runGin() {
	r := gin.Default()
	serverConf, err := helper.GetServerConf()
	if err != nil {
		return
	}
	// 用户登录
	r.POST("/login", service.Login)

	r.Run(serverConf.Web.Port)
}
