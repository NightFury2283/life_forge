package core_http_server

import (
	"fmt"
	"net/http"
)

type ApiVersion string

var (
	ApiVersion1 = ApiVersion("v1")
	ApiVersion2 = ApiVersion("v2")
)

type APIVersionRouter struct {
	*http.ServeMux
	apiVersion ApiVersion
}

func NewAPIVersionRouter(
	apiVersion ApiVersion,
) *APIVersionRouter {
	return &APIVersionRouter{
		ServeMux:   http.NewServeMux(),
		apiVersion: apiVersion,
	}
}

func (r *APIVersionRouter) RegisterRoutes(routes ...Route) {
	for _, route := range routes {
		pattern := fmt.Sprintf("%s %s", route.Method, route.Path)

		r.Handle(pattern, route.Handler)
	}
}

func (r *APIVersionRouter) RegisterGroup(group RouteGroup) {
	for _, route := range group.Routes {
		var handler http.Handler = route.Handler

		for i := len(route.Middleware) - 1; i >= 0; i-- {
			handler = route.Middleware[i](handler)
		}

		for i := len(group.Middleware) - 1; i >= 0; i-- {
			handler = group.Middleware[i](handler)
		}

		pattern := fmt.Sprintf("%s %s", route.Method, route.Path)
		r.Handle(pattern, handler)
	}
}
