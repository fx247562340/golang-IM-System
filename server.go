package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

//server对象
type Server struct {
	Ip string
	Port int

	//在线用户的列表
	OnlineMap map[string]*User
	//同步的锁
	mapLock sync.RWMutex

	//消息广播的channel
	Message chan string
}

//创建一个server的接口
func NewServer(ip string,port int) *Server  {
	server := &Server{
		Ip : ip,
		Port: port,
		OnlineMap: make(map[string]*User),
		Message: make(chan string),
	}
	return server
}
//监听Message广播消息channel的goroutine，一旦有消息就发送给全部在线的User
func (t *Server) ListenMessage()  {
	for true {
		msg := <- t.Message

		//将message发送给全部在线的user
		t.mapLock.Lock()
		for _, cli := range t.OnlineMap{
			cli.C <- msg
		}
		t.mapLock.Unlock()
	}
}


//广播消息的方法
func (t *Server) BroadCast(user *User, msg string)  {
	sendMsg := "[" + user.Addr + "]" + user.Name + ":" + msg

	t.Message <- sendMsg
}

//业务处理
func (t *Server) Handler(conn net.Conn)  {
	//当前链接的业务
	//fmt.Println("链接建立成功")

	user := NewUser(conn, t)
	//用户上线，将用户加入的到onlineMap中
	user.Online()

	//监听用户是否活跃的channel
	isLive := make(chan bool)

	//接受客户发送的消息
	go func() {
		buf := make([]byte, 4896)
		for {
			n,err := conn.Read(buf)
			if n == 0 {
				user.Offline()
				return
			}

			if err != nil && err != io.EOF {
				fmt.Println("Conn Read error:", err)
				return
			}
			//提取用户的消息 去除"\n"
			msg := string(buf[:n-1])

			//将得到的消息进行广播
			user.DoMessage(msg)

			//用户发送任意消息，代表当前用户是活跃的
			isLive <- true
		}
	}()

	//当前handler阻塞
	for true {
		select {
		case <- isLive:
			//当前用户是活跃的，应该重置定时器
			//不做任何处理，为了激活select 更新下面的定时器
		case <- time.After(time.Second * 300):
			//已经超时
			//将当前用户的客户端强制踢出
			user.SendMsg("你被踢出了")

			//销毁用户资源
			close(user.C)
			//关闭连接
			conn.Close()
			//退出当前的handler
			return //或者runtime.Goexit()
		}
	}

}

//启动服务器的接口
func (t *Server) Start()  {
	//socket listen
	listener, err := net.Listen("tcp",fmt.Sprintf("%s:%d",t.Ip,t.Port))
	if err != nil {
		fmt.Println("net.Listen err:",err)
		return
	}
	//close listen socket
	defer listener.Close()

	//启动监听Message的goroutine
	go t.ListenMessage()

	for  {
		//accept
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("listener accept err:", err)
			continue
		}
		//do handler
		go t.Handler(conn)
	}


}
