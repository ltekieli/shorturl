package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"

	"github.com/ltekieli/shorturl/log"
)

const RequestLimit = 2048

type Config struct {
	port    *uint
	static  *string
	apiIp   *string
	apiPort *uint
}

var (
	gConfig Config
	gServer *http.Server
)

type LongLink struct {
	Url string `json:"url"`
}

type ShortLink struct {
	Url string `json:"url"`
}

func resolve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	shortLink := ShortLink{Url: vars["shortid"]}
	b, err := json.Marshal(shortLink)
	if err != nil {
		log.Errorf("Cannot serialize short link: %s", err)
		http.Error(w, "Cannot serialize short link", http.StatusInternalServerError)
		return
	}

	resp, err := http.Post(fmt.Sprintf("http://%s:%d/api/resolve", *gConfig.apiIp, *gConfig.apiPort), "application/json", bytes.NewReader(b))
	if err != nil {
		log.Errorf("Failed to resolve short id: %s", err)
		http.Error(w, "Failed to resolve short id", http.StatusInternalServerError)
		return
	}

	if resp.StatusCode != 200 {
		log.Error("Failed to resolve short id")
		http.Error(w, "Failed to resolve short id", http.StatusInternalServerError)
		return
	}

	reqBody, err := io.ReadAll(io.LimitReader(resp.Body, RequestLimit))
	if err != nil {
		log.Errorf("Failed to read resolve response: %s", err)
		http.Error(w, "Failed to read resolve response", http.StatusInternalServerError)
		return
	}

	var longLink LongLink
	err = json.Unmarshal(reqBody, &longLink)
	if err != nil {
		log.Errorf("Resolve response contains invalid LongLink: %s", err)
		http.Error(w, "Resolve response contains invalid LongLink", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, longLink.Url, http.StatusSeeOther)
}

func startServer(ip string, port uint16) {
	router := mux.NewRouter().StrictSlash(true)
	router.PathPrefix("/static").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir(*gConfig.static)))).Methods("GET")
	router.HandleFunc("/x/{shortid}", resolve).Methods("GET")
	router.PathPrefix("/").Handler(http.FileServer(http.Dir(*gConfig.static))).Methods("GET")

	gServer = &http.Server{Addr: fmt.Sprintf("%s:%d", ip, port), Handler: router}

	go func() {
		log.Info("Serving requests")
		if err := gServer.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("Server shutdown")
			} else {
				log.Errorf("Server error during runtime: %s", err)
				panic(err)
			}
		}
	}()
}

func stopServer() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := gServer.Shutdown(ctx); err != nil {
		log.Errorf("Server error during shutdown: %s", err)
	}
}

func waitForInterrupt() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	func() {
		for {
			select {
			case <-stop:
				return
			}
		}
	}()
}

func getConfig() {
	gConfig = Config{}
	gConfig.port = flag.Uint("port", 8080, "Server port.")
	gConfig.static = flag.String("static", "", "Static content directory. (Required)")
	gConfig.apiIp = flag.String("api-ip", "", "IP of the API server. (Required)")
	gConfig.apiPort = flag.Uint("api-port", 8090, "Port of the API server")
	flag.Parse()

	if *gConfig.static == "" || *gConfig.apiIp == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	getConfig()
	log.Info("Starting URL shortener web...")
	startServer("0.0.0.0", uint16(*gConfig.port))
	defer stopServer()
	waitForInterrupt()
}
