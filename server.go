package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"nchc.org.tw/nchc-web-2018/server/nchc"
)

// JSONIndent indent for json
const JSONIndent = "  "

func runServer() {
	e := echo.New()
	// e.Use(middleware.Logger())
	// e.Use(middleware.CORS())
	// e.Use(middleware.Recover())
	apiUser := nchc.GetVariable("apiUser")
	apiPassword := nchc.GetVariable("apiPassword")

	if apiUser != "" && apiPassword != "" {
		e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
			if username == apiUser && password == apiPassword {
				return true, nil
			}
			return false, nil
		}))
	}

	e.Static("/", "web") // for release version
	e.GET("data-sources", nchc.ReadDataSources)
	e.GET("/records/:name", nchc.ReadRecords)
	e.POST("/load-vibration-images", nchc.LoadVibrationImages)
	e.POST("/load-oven-images", nchc.LoadOvenImages)

	sPort := nchc.GetVariable("servicePort")
	if sPort == "" {
		sPort = "3266"
	}
	sPort = fmt.Sprintf(":%s", sPort)
	s := &http.Server{
		Addr:         sPort,
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Minute,
	}
	s.SetKeepAlivesEnabled(true)
	e.Logger.Fatal(e.StartServer(s))
	//e.Logger.Fatal(e.Start(sPort))
}

func main() {
	runServer()
}
