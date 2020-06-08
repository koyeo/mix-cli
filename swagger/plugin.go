package swagger

import (
	"fmt"
	"github.com/flosch/pongo2"
	"github.com/koyeo/snippet/logger"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/skratchdot/open-golang/open"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

const (
	NAME = "swagger"
)

const version = "1.0"

type Handler struct {
	address     string
	port        string
	swaggerPath string
	swaggerFile string
	url         string
	fileSystem  http.FileSystem
}

func NewHandler() *Handler {
	handler := new(Handler)
	return handler
}

func (p *Handler) Name() string {
	return NAME
}

func (p *Handler) Commands() cli.Commands {

	commands := []cli.Command{
		{
			Name:  "swagger:version",
			Usage: "Show swagger plugin version",
			Action: func(ctx *cli.Context) (err error) {
				fmt.Println(version)
				return
			},
		},
		{
			Name:   "swagger:serve",
			Usage:  "Lunch swagger server",
			Action: p.SwaggerServeCommand,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "address",
					Usage: "Bind address",
				},
				cli.StringFlag{
					Name:  "port",
					Usage: "Bind port",
				},
				cli.BoolFlag{
					Name:  "quiet",
					Usage: "Quiet mode",
				},
				cli.StringFlag{
					Name:  "url",
					Usage: "url address",
				},
			},
		},
	}

	return commands
}

func (p *Handler) parseSwaggerPath(arg string) {
	p.swaggerFile = path.Base(arg)
	p.swaggerPath = strings.TrimSuffix(arg, p.swaggerFile)
	p.fileSystem = http.Dir(p.swaggerPath)
	return
}

func (p *Handler) swaggerContent() (content string, err error) {

	file, err := p.fileSystem.Open(p.swaggerFile)
	if err != nil {
		logger.Error("Open swagger file error: ", err)
		return
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		logger.Error("Read swagger file error: ", err)
		return
	}

	content = string(data)

	return
}

func (p *Handler) SwaggerServeCommand(ctx *cli.Context) (err error) {

	args := ctx.Args()
	if len(args) == 0 {
		logger.Error("Please input swagger file", nil)
		return nil
	} else if len(args) > 1 {
		logger.Error("Only accept one swagger file", nil)
		return nil
	}
	p.parseSwaggerPath(args[0])

	e := echo.New()

	e.Use(middleware.CORS())

	e.GET("/swagger.yaml", p.GetSwaggerYaml)
	e.GET("/", p.GetBrowser)
	e.GET("/:path", p.GetBrowser)

	p.port = ctx.String("port")
	p.address = ctx.String("address")
	p.url = ctx.String("url")

	if strings.TrimSpace(p.address) == "" {
		p.address = "127.0.0.1"
	}
	if strings.TrimSpace(p.port) == "" {
		p.port = "7100"
	}

	if !ctx.Bool("quiet") {
		go func() {
			time.Sleep(500 * time.Millisecond)
			err = open.Run(fmt.Sprintf("http://%s:%s", p.address, p.port))
			if err != nil {
				logger.Error("Open swagger browser error", err)
			}
		}()
	}
	e.HideBanner = true
	err = e.Start(fmt.Sprintf("%s:%s", p.address, p.port))
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (p *Handler) GetBrowser(c echo.Context) error {

	path := c.Param("path")
	if strings.TrimSpace(path) == "" {
		path = "/index.html"
	}
	path = filepath.Join("/", path)
	handle, err := Browser.Open(path)
	if err != nil {
		logger.Error(fmt.Sprintf(`Read template error`), err)
		return err
	}
	defer func() {
		_ = handle.Close()
	}()

	bs, err := ioutil.ReadAll(handle)
	if err != nil {
		logger.Error(fmt.Sprintf(`Read template error`), err)
		return err
	}

	content := string(bs)
	if path == "/index.html" {
		url := fmt.Sprintf("http://%s:%s/swagger.yaml", p.address, p.port)
		if strings.TrimSpace(p.url) != "" {
			url = p.url
		}
		content, err = pongo2.RenderTemplateString(string(bs), pongo2.Context{
			"SwaggerYamlUrl": url,
		})
		if err != nil {
			return err
		}
	}

	if strings.HasSuffix(path, ".html") {
		c.Response().Header().Set("Content-type", "text/html")
	}

	if strings.HasSuffix(path, ".css") {
		c.Response().Header().Set("Content-type", "text/css")
	}

	if strings.HasSuffix(path, ".js") {
		c.Response().Header().Set("Content-type", "text/javascript")
	}

	return c.String(http.StatusOK, content)
}

func (p *Handler) GetSwaggerYaml(c echo.Context) error {

	content, err := p.swaggerContent()
	if err != nil {
		return err
	}

	return c.String(http.StatusOK, content)
}
