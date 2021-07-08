package main

import (
	"net"
	"strings"
)

type User struct {
	Name string
	Addr string
	C chan string
	conn net.Conn
	server *Server
}

//创建一个user的API
func NewUser(conn net.Conn, server *Server)  *User {
	userAddr := conn.RemoteAddr().String()
	user := &User{
		Name : userAddr,
		Addr: userAddr,
		C: make(chan string),
		conn: conn,
		server: server,
	}
	//启动监听当前user channel消息的goroutine
	go user.ListenMessage()

	return user
}

//用户的上线业务
func (t *User) Online()  {
	//用户上线，将用户加入的到onlineMap中
	t.server.mapLock.Lock()
	t.server.OnlineMap[t.Name] = t
	t.server.mapLock.Unlock()

	//广播当前用户上线消息
	t.server.BroadCast(t,"已上线")
}

//用户的下线业务
func (t *User) Offline()  {
	//用户下线，将用户从onlineMap中删除
	t.server.mapLock.Lock()
	delete(t.server.OnlineMap, t.Name)
	t.server.mapLock.Unlock()

	//广播当前用户上线消息
	t.server.BroadCast(t,"下线")
}

//给当前user对应的客户端发送消息
func (t *User) SendMsg(msg string)  {
	t.conn.Write([]byte(msg))
}

//用户处理消息的业务
func (t *User) DoMessage(msg string)  {
	if msg == "who" {
		//查询当前在线用户的指令
		t.server.mapLock.Lock()
		for _, user := range t.server.OnlineMap{
			onlineMsg := "[" + user.Addr + "]" +user.Name + ":" + "在线...\n"
			t.SendMsg(onlineMsg)
		}
		t.server.mapLock.Unlock()
	}else if len(msg) > 7 && msg[:7] == "rename|"{
		//消息格式rename|张三
		newName := strings.Split(msg,"|")[1]
		//判断newName是否存在
		_, ok := t.server.OnlineMap[newName]
		if ok {
			t.SendMsg("当前用户名已被使用")
		}else {
			t.server.mapLock.Lock()
			delete(t.server.OnlineMap,t.Name)
			t.server.OnlineMap[newName] = t
			t.server.mapLock.Unlock()

			t.Name = newName
			t.SendMsg("您已更新用户名：" + t.Name + "\n")
		}
	} else if len(msg) > 4 && msg[:3] == "to|" {
		//消息格式 to|张三|消息内容

		//获取对方用户名
		remoteName := strings.Split(msg,"|")[1]
		if remoteName == "" {
			t.SendMsg("消息格式不正确，请使用 \"to|张三|你好啊\" 格式。\n")
			return
		}
		//根据用户名 得到对方user对象
		remoteUser, ok := t.server.OnlineMap[remoteName]
		if !ok {
			t.SendMsg("该用户名不存在\n")
			return
		}
		//获取消息内容 通过对方的user对象 将消息内容发送过去
		content := strings.Split(msg,"|")[2]
		if content == "" {
			t.SendMsg("无消息内容，请重新输入\n")
			return
		}
		remoteUser.SendMsg(t.Name + "对您说:" + content)
	}else {
		t.server.BroadCast(t, msg)
	}
}

//监听当前User channel的方法,一旦有消息 就发送给客户端
func (t *User) ListenMessage()  {
	for true {
		msg := <-t.C

		t.conn.Write([] byte(msg + "\n"))
	}
}