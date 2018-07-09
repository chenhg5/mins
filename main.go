package main

import (
	"runtime"
	"fmt"
	"github.com/larspensjo/config"
	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
	"errors"
	"log"
	"github.com/json-iterator/go"
	"github.com/mgutz/ansi"
	"runtime/debug"
	"strconv"
	"os"
	"os/signal"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	InitDB("root", "root", "3306", "localhost", "example")

	fmt.Println("Hello Mins")

	// param:
	// config.ini

	router := fasthttprouter.New()

	router.GET("/resource/:table", GetResources)
	router.DELETE("/resource/:table/id/:id", DeleteResources)
	router.PUT("/resource/:table/id/:id", ModifyResources)
	router.POST("/resource/:table", NewResources)
	router.NotFound = NotFoundHandle

	go func() {
		fasthttp.ListenAndServe(":4006", router.Handler)
	}()

	osSignals := make(chan os.Signal)
	signal.Notify(osSignals, os.Interrupt)

	<-osSignals

	fmt.Println("")
	fmt.Println("Bye bye~~")
}

func GetResources(ctx *fasthttp.RequestCtx)  {

	defer handle(ctx)

	table := ctx.UserValue("table").(string)
	resource, _ := Query("select * from " + table)

	jsonByte, err := json.Marshal(resource)

	if err != nil {
		ctx.Error(`{"code":500, "msg":"json marshal error"}`, fasthttp.StatusInternalServerError)
	}

	ctx.WriteString(`{"code":200, "msg":"ok", "data": ` + string(jsonByte[:]) + `}`)
}

func NewResources(ctx *fasthttp.RequestCtx)  {
	defer handle(ctx)
}

func DeleteResources(ctx *fasthttp.RequestCtx)  {
	defer handle(ctx)
	table := ctx.UserValue("table").(string)
	id := string(ctx.QueryArgs().Peek("id")[:])
	Exec("delete from " + table + " where id = ?", id)
	ctx.WriteString(`{"code":200, "msg":"ok"}`)
}

func ModifyResources(ctx *fasthttp.RequestCtx)  {
	defer handle(ctx)
}

func NotFoundHandle(ctx *fasthttp.RequestCtx)  {
	defer handle(ctx)
	ctx.Error(`{"code":404, "msg":"route not found"}`, fasthttp.StatusNotFound)
}

// 全局错误处理
func handle(ctx *fasthttp.RequestCtx) {

	log.Println("[GoAdmin]",
		ansi.Color(" "+strconv.Itoa(ctx.Response.StatusCode())+" ", "white:blue"),
		ansi.Color(" "+string(ctx.Method()[:])+"   ", "white:blue+h"),
		string(ctx.Path()))

	if err := recover(); err != nil {
		fmt.Println(err)
		fmt.Println(string(debug.Stack()[:]))
		ctx.Error(`{"code":500, "msg":"系统错误"}`, fasthttp.StatusInternalServerError)
		return
	}
}

func GetConfig(file string, sec string) (map[string]string, error) {
	targetConfig := make(map[string]string)
	cfg, err := config.ReadDefault(file)
	if err != nil {
		return targetConfig, errors.New("unable to open config file or wrong fomart")
	}
	sections := cfg.Sections()
	if len(sections) == 0 {
		return targetConfig, errors.New("no " + sec + " config")
	}
	for _, section := range sections {
		if section != sec {
			continue
		}
		sectionData, _ := cfg.SectionOptions(section)
		for _, key := range sectionData {
			value, err := cfg.String(section, key)
			if err == nil {
				targetConfig[key] = value
			}
		}
		break
	}
	return targetConfig, nil
}