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

	if len(os.Args) > 1 {
		if os.Args[1] == "version" || os.Args[1] == "--v" || os.Args[1] == "-v" {
			fmt.Println("0.0.4")
			os.Exit(-1)
		}
		if os.Args[1] == "help" || os.Args[1] == "--h" || os.Args[1] == "-h" {
			fmt.Println(" __  __ _")
			fmt.Println(`|  \/  (_)_ __  ___`)
			fmt.Println(`| |\/| | | '_ \/ __|`)
			fmt.Println(`| |  | | | | | \__ \`)
			fmt.Println(`|_|  |_|_|_| |_|___/`)
			fmt.Printf("\n")
			fmt.Printf("usage: \n")
			fmt.Printf("    --h|-h|help         get help message \n")
			fmt.Printf("    --v|-v|version      get current version \n")
			fmt.Printf("    --c                 config-file-path \n")
			fmt.Printf("    --p                 server-port, 4006 default \n")
			fmt.Printf("\n")
			fmt.Printf("api route: \n")
			fmt.Printf("    GET /{table}/{id}\n")
			fmt.Printf("    PUT /{table}/{id}\n")
			fmt.Printf("    DELETE /{table}/{id}\n")
			fmt.Printf("    POST /{table}\n")
			fmt.Printf("\n")
			os.Exit(-1)
		}
	}

	//fmt.Println("Hello Mins")

	var configFile string
	var port string
	flag.StringVar(&configFile, "c", "", "config path")
	flag.StringVar(&port, "p", "4006", "server port")
	flag.Parse()

	router := fasthttprouter.New()

	if configFile != "" {
		databseCfg, configErr := GetConfig(configFile, "database")

		if configErr != nil {
			panic(configErr)
		}

		InitDB(databseCfg["user"], databseCfg["password"], databseCfg["port"], databseCfg["addr"], databseCfg["database"])

		router.GET("/:table/:id", GetResources)
		router.DELETE("/:table/:id", DeleteResources)
		router.PUT("/:table/:id", ModifyResources)
		router.POST("/:table", NewResources)

		severCfg, _ := GetConfig(configFile, "server")
		port = severCfg["port"]
	}

	fmt.Println(" __  __ _")
	fmt.Println(`|  \/  (_)_ __  ___`)
	fmt.Println(`| |\/| | | '_ \/ __|`)
	fmt.Println(`| |  | | | | | \__ \`)
	fmt.Println(`|_|  |_|_|_| |_|___/`)
	fmt.Printf("\n")
	fmt.Printf("listening on port %s, ", port)

	if configFile != "" {
		fmt.Printf("serve for rest api. \n\n")
		fmt.Printf("available route: \n")
		fmt.Printf("    GET    /{table}/{id}\n")
		fmt.Printf("    PUT    /{table}/{id}\n")
		fmt.Printf("    DELETE /{table}/{id}\n")
		fmt.Printf("    POST   /{table}\n")
	} else {
		fmt.Printf("serve for static file. \n")
	}

	fmt.Printf("\n")

	router.NotFound = NotFoundHandler

	go func() {
		fasthttp.ListenAndServe(":"+port, router.Handler)
	}()

	osSignals := make(chan os.Signal)
	signal.Notify(osSignals, os.Interrupt)

	<-osSignals

	fmt.Println("")
	fmt.Println("Bye bye~~")
}

func PathExist(_path string) bool {
	_, err := os.Stat(_path)
	if err != nil && os.IsNotExist(err) {
		return false
	}
	return true
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

	Exec("insert into "+table+"("+fieldStr[0:len(fieldStr)-1]+") "+"values ("+quesStr[0:len(quesStr)-1]+")", valueArr...)

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

	Exec("update "+table+" set "+fieldStr[0:len(fieldStr)-1]+" where id = ?", valueArr...)

	ctx.SetContentType("application/json")
	ctx.WriteString(`{"code":200, "msg":"ok"}`)
}

var fs = &fasthttp.FS{
	Root:               "./",
	IndexNames:         []string{"index.html"},
	GenerateIndexPages: true,
	Compress:           false,
	AcceptByteRange:    false,
}

var FSHandler = fs.NewRequestHandler()

func NotFoundHandler(ctx *fasthttp.RequestCtx) {
	defer handle(ctx)

	if PathExist(string(ctx.Path())) {
		FSHandler(ctx)
	} else {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetContentType("application/json")
		ctx.WriteString(`{"code":404, "msg":"route not found"}`)
	}
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

// 全局错误处理
func handle(ctx *fasthttp.RequestCtx) {

	log.Println("[Mins]",
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
