package main

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/net/websocket"
)

type Store struct {
	value  string
	log    []string
	offset int
	cond   *sync.Cond
}

func NewStore() *Store {
	return &Store{"hello world", nil, 0, sync.NewCond(&sync.Mutex{})}
}

func (s *Store) pushLog(action, message string) {
	s.log = append(s.log, "<li>"+time.Now().Format("2006/01/02 15:04:05 MST")+" <b>"+action+"</b> "+template.HTMLEscapeString(message))

	for len(s.log) > 10 {
		s.log = s.log[1:]
		s.offset += 1
	}

	s.cond.Broadcast()
}

func (s *Store) Set(value string) {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	s.pushLog("SET", s.value+" -> "+value)
	s.value = value
}

func (s *Store) Get() string {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	s.pushLog("GET", s.value)
	return s.value
}

func (s *Store) Log() string {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()

	return "<ol start=" + strconv.Itoa(s.offset+1) + ">" + strings.Join(s.log, "") + "</ol>"
}

func (s *Store) WaitNewEntry() string {
	s.cond.L.Lock()
	defer s.cond.L.Unlock()
	s.cond.Wait()

	return s.log[len(s.log)-1]
}

func main() {
	e := echo.New()
	s := NewStore()

	header := "<!DOCTYPE html><html lang=en><title>settable-web</title><a href=/set>set</a> | <a href=/get>get</a> | <a href=/log>log</a><hr>"

	setPage := func(c echo.Context) error {
		return c.HTML(http.StatusOK, header+"current value: "+template.HTMLEscapeString(s.Get())+"<form action=/set method=post><input name=value autofocus /><input type=submit /></form>")
	}

	e.GET("/set", setPage)
	e.POST("/set", func(c echo.Context) error {
		s.Set(c.FormValue("value"))
		return setPage(c)
	})

	e.GET("/get", func(c echo.Context) error {
		return c.String(http.StatusOK, s.Get())
	})

	e.GET("/log", func(c echo.Context) error {
		return c.HTML(http.StatusOK, header+s.Log()+"<script>new WebSocket(`ws://${location.host}/log/ws`).onmessage = (ev) => document.querySelector('ol').innerHTML += ev.data</script>")
	})
	e.GET("/log/ws", echo.WrapHandler(websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()

		for {
			if _, err := ws.Write([]byte(s.WaitNewEntry())); err != nil {
				break
			}
		}
	})))

	e.Use(middleware.Logger(), middleware.Recover())
	e.Logger.Fatal(e.Start(":8080"))
}
