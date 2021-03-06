package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/mchmarny/gcputil/env"
	"go.opencensus.io/plugin/ochttp"
	"gopkg.in/olahol/melody.v1"
)

const (
	traceExporterNotSet = "trace-not-set"
)

var (
	logger = log.New(os.Stdout, "VIEWER == ", 0)

	// AppVersion will be overritten during build
	AppVersion = "v0.0.1-default"

	// service
	servicePort = env.MustGetEnvVar("PORT", "8083")
	sourceTopic = env.MustGetEnvVar("VIEWER_SOURCE_TOPIC_NAME", "processed")
	exporterURL = env.MustGetEnvVar("TRACE_EXPORTER_URL", traceExporterNotSet)

	broadcaster *melody.Melody
)

func main() {
	gin.SetMode(gin.ReleaseMode)

	// router
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(Options)

	// ws
	broadcaster = melody.New()
	broadcaster.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// static
	r.LoadHTMLGlob("resource/template/*")
	r.Static("/static", "./resource/static")
	r.StaticFile("/favicon.ico", "./resource/static/img/favicon.ico")

	// simple routes
	r.GET("/", rootHandler)
	r.GET("/dapr/subscribe", subscribeHandler)

	// websockets
	r.GET("/ws", func(c *gin.Context) {
		broadcaster.HandleRequest(c.Writer, c.Request)
	})

	// topic route
	viewerRoute := fmt.Sprintf("/%s", sourceTopic)
	logger.Printf("viewer route: %s", viewerRoute)
	r.POST(viewerRoute, eventHandler)

	// server
	hostPort := net.JoinHostPort("0.0.0.0", servicePort)
	logger.Printf("Server (%s) starting: %s \n", AppVersion, hostPort)
	if err := http.ListenAndServe(hostPort, &ochttp.Handler{Handler: r}); err != nil {
		logger.Fatalf("server error: %v", err)
	}
}

// Options midleware
func Options(c *gin.Context) {
	if c.Request.Method != "OPTIONS" {
		c.Next()
	} else {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "POST,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "authorization, origin, content-type, accept")
		c.Header("Allow", "POST,OPTIONS")
		c.Header("Content-Type", "application/json")
		c.AbortWithStatus(http.StatusOK)
	}
}
