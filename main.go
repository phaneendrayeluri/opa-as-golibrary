package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/open-policy-agent/opa/plugins"
	"github.com/open-policy-agent/opa/server"
	"github.com/open-policy-agent/opa/storage/inmem"
)

const (
	// OPAHost Address
	OPAHost = "http://localhost:8084"
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

		opa, err := server.New().WithInsecureAddress(OPAHost).WithStore(inmemstore).WithManager(manager).Init(ctx)
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

	if err := createPolicies(); err != nil {
		log.Fatal("error creating policies, err - ", err)
	}

	// Start APP Server
	http.HandleFunc("/fetch", opaMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`{"message":"hello, %s"}`, r.URL.Query().Get("user"))))
	}))
	http.HandleFunc("/update", opaMiddleware(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		w.Write([]byte("OK"))
	}))
	http.ListenAndServe(":8080", nil)
}

func createPolicies() error {
	bees, err := ioutil.ReadFile("policies")
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPut, OPAHost+"/v1/policies/http/authz", bytes.NewBuffer(bees))
	if err != nil {
		return err
	}

	ctx, cnf := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cnf()

	req = req.WithContext(ctx)
	req.Header.Set("Content-Type", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("policy create/update unsuccessful, StatusCode: %d", resp.StatusCode)
	}

	return nil
}

func opaMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		user := r.URL.Query().Get("user")
		if user = strings.TrimSpace(user); user == "" {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"message": "missing required query param user"}`))
			return
		}

		payload := map[string]interface{}{
			"input": map[string]string{
				"method": r.Method,
				"path":   r.URL.Path,
				"user":   user,
			},
		}

		bees, err := json.Marshal(payload)
		if err != nil {
			log.Println("json marshal failure, err - ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		req, err := http.NewRequest(http.MethodPost, OPAHost+"/v1/data/http/authz", bytes.NewBuffer(bees))
		if err != nil {
			log.Println("error creating http request, err - ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		ctx, cnf := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cnf()

		req = req.WithContext(ctx)
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Println("error during decision request, err - ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != http.StatusOK {
			log.Println("unsuccessful response from opa server, err - ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var respPayload decisionResponse
		if err := json.NewDecoder(resp.Body).Decode(&respPayload); err != nil {
			log.Println("unsuccessful unmarshal of opa decision response, err - ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if !respPayload.Result.Allow {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

type decisionResponse struct {
	Result result `json:"result"`
}
type result struct {
	Allow bool `json:"allow"`
}
