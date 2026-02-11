package chi

// Route represents a module that can register HTTP routes.
// Modules implement this interface to register their routes with the server.
type Route interface {
	Setup(server *Server)
}

// RouteRegistry manages a collection of routes and sets them up on the server.
type RouteRegistry struct {
	routes []Route
}

// NewRouteRegistry creates a new route registry.
func NewRouteRegistry() *RouteRegistry {
	return &RouteRegistry{routes: make([]Route, 0)}
}

// Add adds a route to the registry.
func (r *RouteRegistry) Add(route Route) {
	r.routes = append(r.routes, route)
}

// SetupAll calls Setup on all registered routes.
func (r *RouteRegistry) SetupAll(server *Server) {
	for _, route := range r.routes {
		route.Setup(server)
	}
}
