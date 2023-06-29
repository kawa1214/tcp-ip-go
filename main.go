package main

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/kawa1214/tcp-ip-go/application"
)

type Todo struct {
	Id        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
}

type CreateTodoRequest struct {
	Title string `json:"title"`
}

type todoList struct {
	todos []Todo
	lock  sync.Mutex
}

func newTodoList() *todoList {
	return &todoList{
		todos: make([]Todo, 0),
	}
}

func (l *todoList) add(title string) Todo {
	l.lock.Lock()
	defer l.lock.Unlock()
	todo := Todo{
		Id:        len(l.todos) + 1,
		Title:     title,
		CreatedAt: time.Now(),
	}
	l.todos = append(l.todos, todo)
	return todo
}

func (l *todoList) getAll() []Todo {
	l.lock.Lock()
	defer l.lock.Unlock()
	return l.todos
}

func main() {
	todoList := newTodoList()
	todoList.add("todo1")

	s := application.NewServer()
	defer s.Close()
	s.ListenAndServe()

	for {
		conn, err := s.Accept()
		if err != nil {
			log.Printf("accept error: %s", err)
			continue
		}

		reqRaw := string(conn.Pkt.Packet.Buf[conn.Pkt.IpHeader.IHL*4+conn.Pkt.TcpHeader.DataOff*4:])
		req, err := application.ParseHttpRequest(reqRaw)
		if err != nil {
			resp := application.NewHttpResponse(application.HttpStatusInternalServerError, err.Error())
			s.Write(conn, resp)
			continue
		}

		log.Printf("request: %v", req)
		if req.Method == "GET" && req.URI == "/todos" {
			todos := todoList.getAll()
			body, err := json.Marshal(todos)
			if err != nil {
				resp := application.NewHttpResponse(application.HttpStatusInternalServerError, err.Error())
				s.Write(conn, resp)
				continue
			}
			resp := application.NewHttpResponse(application.HttpStatusCreated, string(body)+string('\n'))
			s.Write(conn, resp)
		}

		if req.Method == "POST" && req.URI == "/todos" {
			var createReq CreateTodoRequest
			err := json.Unmarshal([]byte(req.Body), &createReq)
			if err != nil {
				resp := application.NewHttpResponse(application.HttpStatusInternalServerError, err.Error())
				s.Write(conn, resp)
				continue
			}
			todo := todoList.add(createReq.Title)
			body, err := json.Marshal(todo)
			if err != nil {
				resp := application.NewHttpResponse(application.HttpStatusInternalServerError, err.Error())
				s.Write(conn, resp)
				continue
			}

			resp := application.NewHttpResponse(application.HttpStatusOK, string(body)+string('\n'))
			s.Write(conn, resp)
		}
	}
}
