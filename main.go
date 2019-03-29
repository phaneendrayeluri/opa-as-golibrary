package main

import (
	"context"
	"log"
	"net/http"

	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/server"
	"github.com/open-policy-agent/opa/storage/inmem"
)

func main() {

	// Start OPA Server
	go func() {

		ctx, cnf := context.WithCancel(context.Background())
		defer cnf()

		inmemstore := inmem.New()

		manager, err := plugins.New(nil, "", inmemstore)
		if err != nil {
			log.Fatal("error initializing opa plugin manager ", err)
		}
		if err := manager.Start(ctx); err != nil {
			log.Fatal("error initializing manager ", err)
		}

		opa, err := server.New().WithInsecureAddress("http://localhost:8084").WithStore(inmemstore).WithManager(manager).Init(ctx)
		if err != nil {
			log.Fatal("error initializing opa ", err)
		}

		loops, err := opa.Listeners()
		if err != nil {
			log.Fatal("error initializing opa listners ", err)
		}
		for _, loop := range loops {
			go func(loop func() error) {
				log.Println("loop terminated, err - ", loop())
			}(loop)
		}

		log.Println("opa started listening")
	}()

	// Start APP Server
	http.HandleFunc("/fetch", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"helloworld"}`))
	})
	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	http.ListenAndServe(":8080", nil)
}
