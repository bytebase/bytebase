package server

import (
	"net/http"
	"net/http/pprof"
	"sync/atomic"

	"github.com/labstack/echo/v5"
)

func registerPprof(e *echo.Echo, runtimeDebug *atomic.Bool) {
	router := e.Group("/debug/pprof")
	handler := func(h http.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if !runtimeDebug.Load() {
				return echo.NewHTTPError(echo.ErrNotFound.StatusCode(), "")
			}
			h.ServeHTTP(c.Response(), c.Request())
			return nil
		}
	}
	router.GET("/", handler(pprof.Index))
	router.GET("/allocs", handler(pprof.Handler("allocs").ServeHTTP))
	router.GET("/block", handler(pprof.Handler("block").ServeHTTP))
	router.GET("/cmdline", handler(pprof.Cmdline))
	router.GET("/goroutine", handler(pprof.Handler("goroutine").ServeHTTP))
	router.GET("/heap", handler(pprof.Handler("heap").ServeHTTP))
	router.GET("/mutex", handler(pprof.Handler("mutex").ServeHTTP))
	router.GET("/profile", handler(pprof.Profile))
	router.POST("/symbol", handler(pprof.Symbol))
	router.GET("/symbol", handler(pprof.Symbol))
	router.GET("/threadcreate", handler(pprof.Handler("threadcreate").ServeHTTP))
	router.GET("/trace", handler(pprof.Trace))
}
