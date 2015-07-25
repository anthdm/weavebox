# weavebox [![GoDoc](https://godoc.org/github.com/twanies/weavebox?status.svg)](https://godoc.org/github.com/twanies/weavebox) [![Travis CI](https://travis-ci.org/twanies/weavebox.svg?branch=master)](https://travis-ci.org/twanies/weavebox)
Opinion based minimalistic web framework for the Go programming language.

## Installation
`go get github.com/twanies/weavebox`

## Features
- fast route dispatching backed by httprouter
- easy to add middleware handlers
- subrouting with seperated middleware handlers
- central based error handling
- build in template engine
- fast, lightweight and extendable

## Basic usage
    package main
    import "github.com/twanies/weavebox"

    func main() {
        app := weavebox.New()

        app.Get("/foo", fooHandler)
        app.Post("/bar", barHandler)
        app.Use(middleware1, middleware2)

        friends := app.Subrouter("/friends")
        friends.Get("/profile", profileHandler)
        friends.Use(middleware3, middleware4)
        
        app.Serve(8080)
    }
More complete examples can be found in the examples folder

## Routes
    app := weavebox.New()

    app.Get("/", func(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
       .. do something .. 
    })
    app.Post("/", func(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
       .. do something .. 
    })
    app.Put("/", func(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
       .. do something .. 
    })
    app.Delete("/", func(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
       .. do something .. 
    })

get named url parameters

    app.Get("/hello/:name", func(ctx *weavebox.Context, w http.ResponseWriter, r *http.Request) error {
        name := ctx.Param("name")
    })

## Static files
    app := weavebox.New()
    app.Static("/assets", "public/assets")

Now our assets are accessable trough /assets/styles.css

## Handlers

## Context

## View / Templates

## Logging

## Helpers


