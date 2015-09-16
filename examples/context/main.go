package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/net/context"

	"github.com/twanies/weavebox"
)

// Simpel example how to use weavebox with a "datastore" by making use
// of weavebox.Context to pass information between middleware and handlers

func main() {
	listen := flag.Int("listen", 3000, "listen address of the application")
	flag.Parse()

	app := weavebox.New()

	// centralizing our errors returned from middleware and request handlers
	app.SetErrorHandler(errorHandler)

	app.Get("/hello/:name", greetingHandler)
	app.Use(dbContextHandler)

	// make a subrouter and register some middleware for it
	admin := app.Box("/admin")
	admin.Get("/:name", adminGreetingHandler)
	admin.Use(authenticate)

	log.Fatal(app.Serve(*listen))
}

type datastore struct {
	name string
}

type dbContext struct {
	context.Context
	ds *datastore
}

func (c *dbContext) Value(key interface{}) interface{} {
	if key == "datastore" {
		return c.ds
	}
	return c.Context.Value(key)
}

func newDatastoreContext(parent context.Context, ds *datastore) context.Context {
	return &dbContext{parent, ds}
}

func dbContextHandler(next weavebox.Handler) weavebox.Handler {
	return func(c *weavebox.Context) error {
		db := datastore{"mydatabase"}
		c.Context = newDatastoreContext(c.Context, &db)
		return next(c)
	}
}

// Only the powerfull have access to the admin routes
func authenticate(next weavebox.Handler) weavebox.Handler {
	return func(c *weavebox.Context) error {
		admins := []string{"toby", "master iy", "c.froome"}
		name := c.Param("name")

		for _, admin := range admins {
			if admin != name {
				return errors.New("access forbidden")
			}
		}
		return next(c)
	}
}

// context helper function to stay lean and mean in your handlers
func datastoreFromContext(ctx context.Context) *datastore {
	return ctx.Value("datastore").(*datastore)
}

func greetingHandler(ctx *weavebox.Context) error {
	name := ctx.Param("name")
	db := datastoreFromContext(ctx.Context)
	greeting := fmt.Sprintf("Greetings, %s\nYour database %s is ready", name, db.name)
	return ctx.Text(http.StatusOK, greeting)
}

func adminGreetingHandler(ctx *weavebox.Context) error {
	name := ctx.Param("name")
	db := datastoreFromContext(ctx.Context)
	greeting := fmt.Sprintf("Greetings powerfull admin, %s\nYour database %s is ready", name, db.name)
	return ctx.Text(http.StatusOK, greeting)
}

// custom centralized error handling
func errorHandler(ctx *weavebox.Context, err error) {
	http.Error(ctx.Response(), "Hey some error occured: "+err.Error(), http.StatusInternalServerError)
}
