package router

import (
	"net/http"

	"github.com/go-sum/foundry/pkg/web"
)

// ResourceHandlers holds optional handlers for a CRUD resource collection.
// Nil fields are skipped — only non-nil handlers are registered.
type ResourceHandlers struct {
	Index   web.Handler // GET    /path
	New     web.Handler // GET    /path/new
	Create  web.Handler // POST   /path
	Show    web.Handler // GET    /path/{id}
	Edit    web.Handler // GET    /path/{id}/edit
	Update  web.Handler // PATCH  /path/{id}
	Destroy web.Handler // DELETE /path/{id}
}

// Resources returns route nodes for a standard CRUD collection resource.
// Route names follow the convention: name + "." + action.
//
// Routes registered (when the corresponding handler is non-nil):
//
//	GET    path           → name.index
//	GET    path/new       → name.new
//	POST   path           → name.create
//	GET    path/{id}      → name.show
//	GET    path/{id}/edit → name.edit
//	PATCH  path/{id}      → name.update
//	DELETE path/{id}      → name.destroy
func Resources(path, name string, h ResourceHandlers) []Node {
	var nodes []Node
	if h.Index != nil {
		nodes = append(nodes, RouteNode(http.MethodGet, path, name+".index", h.Index))
	}
	if h.New != nil {
		nodes = append(nodes, RouteNode(http.MethodGet, path+"/new", name+".new", h.New))
	}
	if h.Create != nil {
		nodes = append(nodes, RouteNode(http.MethodPost, path, name+".create", h.Create))
	}
	if h.Show != nil {
		nodes = append(nodes, RouteNode(http.MethodGet, path+"/{id}", name+".show", h.Show))
	}
	if h.Edit != nil {
		nodes = append(nodes, RouteNode(http.MethodGet, path+"/{id}/edit", name+".edit", h.Edit))
	}
	if h.Update != nil {
		nodes = append(nodes, RouteNode(http.MethodPatch, path+"/{id}", name+".update", h.Update))
	}
	if h.Destroy != nil {
		nodes = append(nodes, RouteNode(http.MethodDelete, path+"/{id}", name+".destroy", h.Destroy))
	}
	return nodes
}

// SingleResourceHandlers holds optional handlers for a singular resource (no collection, no ID).
// Nil fields are skipped — only non-nil handlers are registered.
type SingleResourceHandlers struct {
	Show    web.Handler // GET    /path
	New     web.Handler // GET    /path/new
	Create  web.Handler // POST   /path
	Edit    web.Handler // GET    /path/edit
	Update  web.Handler // PATCH  /path
	Destroy web.Handler // DELETE /path
}

// Resource returns route nodes for a singular resource (no collection index, no ID in path).
// Route names follow the convention: name + "." + action.
//
// Routes registered (when the corresponding handler is non-nil):
//
//	GET    path      → name.show
//	GET    path/new  → name.new
//	POST   path      → name.create
//	GET    path/edit → name.edit
//	PATCH  path      → name.update
//	DELETE path      → name.destroy
func Resource(path, name string, h SingleResourceHandlers) []Node {
	var nodes []Node
	if h.Show != nil {
		nodes = append(nodes, RouteNode(http.MethodGet, path, name+".show", h.Show))
	}
	if h.New != nil {
		nodes = append(nodes, RouteNode(http.MethodGet, path+"/new", name+".new", h.New))
	}
	if h.Create != nil {
		nodes = append(nodes, RouteNode(http.MethodPost, path, name+".create", h.Create))
	}
	if h.Edit != nil {
		nodes = append(nodes, RouteNode(http.MethodGet, path+"/edit", name+".edit", h.Edit))
	}
	if h.Update != nil {
		nodes = append(nodes, RouteNode(http.MethodPatch, path, name+".update", h.Update))
	}
	if h.Destroy != nil {
		nodes = append(nodes, RouteNode(http.MethodDelete, path, name+".destroy", h.Destroy))
	}
	return nodes
}
