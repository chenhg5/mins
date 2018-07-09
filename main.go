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
	"github.com/go-sql-driver/mysql"
	"flag"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	fmt.Println("Hello Mins")

	var configFile string
	flag.StringVar(&configFile, "c", "./config.ini", "config path")
	flag.Parse()

	databseCfg, configErr := GetConfig(configFile, "database")

	if configErr != nil {
		panic(configErr)
	}

	InitDB(databseCfg["user"], databseCfg["password"], databseCfg["port"], databseCfg["addr"], databseCfg["database"])

	router := fasthttprouter.New()

	router.GET("/resource/:table/id/:id", GetResources)
	router.DELETE("/resource/:table/id/:id", DeleteResources)
	router.PUT("/resource/:table/id/:id", ModifyResources)
	router.POST("/resource/:table", NewResources)
	router.NotFound = NotFoundHandle

	go func() {
		severCfg, _ := GetConfig(configFile, "server")
		fasthttp.ListenAndServe(":"+severCfg["port"], router.Handler)
	}()

	osSignals := make(chan os.Signal)
	signal.Notify(osSignals, os.Interrupt)

	<-osSignals

	fmt.Println("")
	fmt.Println("Bye bye~~")
}

func GetResources(ctx *fasthttp.RequestCtx) {

	defer handle(ctx)

	table := ctx.UserValue("table").(string)
	id := ctx.UserValue("id").(string)

	resource, _ := Query("select * from "+table+" where id = ?", id)

	if len(resource) < 1 {
		ctx.SetContentType("application/json")
		ctx.WriteString(`{"code":200, "msg":"ok", "data": {}}`)
	}

	jsonByte, err := json.Marshal(resource[0])

	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetContentType("application/json")
		ctx.WriteString(`{"code":500, "msg":"json marshal error"}`)
	}

	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":200, "msg":"ok", "data": ` + string(jsonByte[:]) + `}`)
}

func NewResources(ctx *fasthttp.RequestCtx) {
	defer handle(ctx)

	table := ctx.UserValue("table").(string)

	columns := GetAllColumns(table)
	fieldStr := ""
	quesStr := ""
	valueArr := make([]interface{}, 0)

	for i := 0; i < len(columns); i++ {
		if value, isIn := IsInFormValue(ctx, columns[i]["Field"].(string)); isIn {
			fieldStr += "`" + columns[i]["Field"].(string) + "`,"
			quesStr += "?,"
			valueArr = append(valueArr, value)
		}
	}

	fieldStr = SliceStr(fieldStr)
	quesStr = SliceStr(quesStr)

	Exec("insert into "+table+"("+fieldStr+") "+"values ("+quesStr+")", valueArr...)

	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":200, "msg":"ok"}`)
}

func DeleteResources(ctx *fasthttp.RequestCtx) {
	defer handle(ctx)
	table := ctx.UserValue("table").(string)
	id := ctx.UserValue("id").(string)
	Exec("delete from "+table+" where id = ?", id)
	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":200, "msg":"ok"}`)
}

func ModifyResources(ctx *fasthttp.RequestCtx) {
	defer handle(ctx)

	table := ctx.UserValue("table").(string)
	id := ctx.UserValue("id").(string)

	columns := GetAllColumns(table)
	fieldStr := ""
	valueArr := make([]interface{}, 0)

	for i := 0; i < len(columns); i++ {
		if value, isIn := IsInFormValue(ctx, columns[i]["Field"].(string)); isIn {
			fieldStr += "`" + columns[i]["Field"].(string) + "` = ?,"
			valueArr = append(valueArr, value)
		}
	}
	valueArr = append(valueArr, id)

	fieldStr = SliceStr(fieldStr)

	Exec("update "+table+" set "+fieldStr+" where id = ?", valueArr...)

	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":200, "msg":"ok"}`)
}

func NotFoundHandle(ctx *fasthttp.RequestCtx) {
	defer handle(ctx)
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":404, "msg":"route not found"}`)
}

func GetAllColumns(table string) []map[string]interface{} {
	colunms, _ := Query("show columns from " + table)
	return colunms
}

func IsInFormValue(ctx *fasthttp.RequestCtx, key string) (string, bool) {
	mf, err := ctx.MultipartForm()
	if err == nil && mf.Value != nil {
		vv := mf.Value[key]
		if len(vv) > 0 {
			return vv[0], true
		} else {
			return "", false
		}
	} else {
		return "", false
	}
}

func SliceStr(s string) string {
	if s == "" {
		return s
	}
	rs := []rune(s)
	length := len(rs)
	return string(rs[0: length-1])
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

		var (
			errMsg     string
			mysqlError *mysql.MySQLError
			ok         bool
		)
		if errMsg, ok = err.(string); ok {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentType("application/json")
			ctx.WriteString(`{"code":500, "msg":"` + errMsg + `"}`)
			return
		} else if mysqlError, ok = err.(*mysql.MySQLError); ok {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentType("application/json")
			ctx.WriteString(`{"code":500, "msg":"` + mysqlError.Error() + `"}`)
			return
		} else {
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			ctx.SetContentType("application/json")
			ctx.WriteString(`{"code":500, "msg":"系统错误"}`)
			return
		}
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
