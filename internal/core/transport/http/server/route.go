package core_http_server

import (
	"net/http"

	core_http_middleware "github.com/NightFury2283/life_forge/internal/core/transport/http/middleware"
)

type Route struct {
	Method     string
	Path       string
	Handler    http.HandlerFunc
	Middleware []core_http_middleware.Middleware
}

func NewRoute(
	method string,
	path string,
	handler http.HandlerFunc,
) Route {
	return Route{
		Method:  method,
		Path:    path,
		Handler: handler,
	}
}

type RouteGroup struct {
	Middleware []core_http_middleware.Middleware
	Routes     []Route
}
