// Package handler is responsible for routes and handling requests
package handler

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Router sets up routes
func Router() (http.Handler, error) {
	router := gin.Default()

	setupHelloRoutes(router)
	return router, nil
}

// InternalRouter configures the internal routes
func InternalRouter() http.Handler {
	router := gin.Default()
	// expose /metrics endpoint via internal server
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	routerGroup1 := router.Group("/internal")
	routerGroup1.GET("/status", func(c *gin.Context) { c.Status(http.StatusOK) })
	return router
}

func setupHelloRoutes(r *gin.Engine) gin.IRoutes {
	return r.GET("/hello", handleHello)
}

func handleHello(c *gin.Context) {
	nameOrDefault := c.DefaultQuery("name", "world")
	c.String(http.StatusOK, "Hello %s", nameOrDefault)
}