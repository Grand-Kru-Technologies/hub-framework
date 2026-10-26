package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/QueerGlobal/hub-framework/api"
	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/QueerGlobal/hub-framework/example/recipe-app/golang/tasks"
)

type rootHandler struct {
	hub http.Handler
}

func (h rootHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		fmt.Fprintf(w, "Welcome to exampleApp!")
		return
	case "/health":
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
		return
	}
	if smokeForwardDisabled() && requestTargetsSmokeForward(r) {
		http.NotFound(w, r)
		return
	}
	if h.hub != nil {
		h.hub.ServeHTTP(w, r)
		return
	}
	http.NotFound(w, r)
}

func smokeForwardDisabled() bool {
	return os.Getenv("ENABLE_SMOKE_FORWARD") != "true"
}

func requestTargetsSmokeForward(r *http.Request) bool {
	svcReq, err := entity.GetRequestFromHttp(r)
	if err != nil {
		return false
	}
	return strings.EqualFold(svcReq.ApiName, "smokeApp") && strings.EqualFold(svcReq.ServiceName, "forward")
}

func main() {
	app := api.NewApplication(
		"exampleApp",
		api.WithLogLevel(api.InfoLevel),
		api.WithPublicPort(8080),
		api.WithHTTPListen(false),
	)
	if app == nil {
		log.Printf("Error creating application")
		return
	}

	exampleTaskConstructor := api.TaskConstructorFunc(func(config map[string]any) (api.Task, error) {
		return tasks.NewExampleTaskGolang(config), nil
	})
	api.RegisterTaskType("exampleTaskGolang", exampleTaskConstructor)

	if err := app.Start(); err != nil {
		log.Printf("Error starting application: %v", err)
		return
	}
	defer app.Stop()

	root := rootHandler{}
	if app.PublicHandler != nil {
		root.hub = app.PublicHandler
	}

	server := &http.Server{Addr: ":8080", Handler: root}

	go func() {
		log.Printf("Starting exampleApp server on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	_ = server.Close()
}
