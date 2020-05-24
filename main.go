package main

import (
	"html/template"
	"io"
	"net/http"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// レイアウト適用済のテンプレートを保存するmap
var templates map[string]*template.Template

// Template はHTMLテンプレートを利用するためのRenderer Interface
type Template struct {
}

// Render はHTMLテンプレートにデータを埋め込んだ結果をWriterに書き込む
func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return templates[name].ExecuteTemplate(w, "layout.html", data)
}

func main() {
	// Echoのインスタンスを生成
	e := echo.New()

	// テンプレートを利用するためのRendererの設定
	t := &Template{}
	e.Renderer = t

	// ミドルウェアを設定
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// 静的ファイルのパスを設定
	e.Static("/public/css/", "./public/css")
	e.Static("/public/js/", "./public/js/")
	e.Static("/public/img/", "./public/img/")

	// 各ルーティングに対するハンドラを設定
	e.GET("/", HandleIndexGet)
	e.GET("/hello", HandleHelloGet)
	e.POST("/hello", HandleHelloPost)
	e.GET("/hello_form", HandleHelloFormGet)
	e.GET("/api/hello", HandleAPIHelloGet)
	e.POST("/api/hello", HandleAPIHelloPost)

	// サーバーを開始
	e.Logger.Fatal(e.Start(":3000"))
}

// 初期化を行います。
func init() {
	loadTemplates()
}

// 各HTMLテンプレートに共通レイアウトを適用した結果を保存します（初期化時に実行）。
func loadTemplates() {
	var baseTemplate = "templates/layout.html"
	templates = make(map[string]*template.Template)
	templates["hello"] = template.Must(
		template.ParseFiles(baseTemplate, "templates/hello.html"))
	templates["hello_form"] = template.Must(
		template.ParseFiles(baseTemplate, "templates/hello_form.html"))
}

// HandleIndexGet は Index のGet時のHTMLデータ生成処理を行います。
func HandleIndexGet(c echo.Context) error {
	return c.Render(http.StatusOK, "hello", "world")
}

// HandleHelloGet は /hello のGet時のHTMLデータ生成処理を行います。
func HandleHelloGet(c echo.Context) error {
	greetingto := c.QueryParam("greetingto")
	return c.Render(http.StatusOK, "hello", greetingto)
}

// HandleHelloPost は /hello のPost時のHTMLデータ生成処理を行います。
func HandleHelloPost(c echo.Context) error {
	greetingto := c.FormValue("greetingto")
	return c.Render(http.StatusOK, "hello", greetingto)
}

// HandleHelloFormGet は /hello_form のGet時のHTMLデータ生成処理を行います。
func HandleHelloFormGet(c echo.Context) error {
	return c.Render(http.StatusOK, "hello_form", nil)
}

// HandleAPIHelloGet は /api/hello のGet時のJSONデータ生成処理を行います。
func HandleAPIHelloGet(c echo.Context) error {
	greetingto := c.QueryParam("greetingto")
	return c.JSON(http.StatusOK, map[string]interface{}{"hello": greetingto})
}

// HelloParam は /api/hello が受けとるJSONパラメータを定義します。
type HelloParam struct {
	GreetingTo string `json:"greetingto"`
}

// HandleAPIHelloPost は /api/hello のPost時のJSONデータ生成処理を行います。
func HandleAPIHelloPost(c echo.Context) error {
	param := new(HelloParam)
	if err := c.Bind(param); err != nil {
		return err
	}
	return c.JSON(http.StatusOK, map[string]interface{}{"hello": param.GreetingTo})
}
