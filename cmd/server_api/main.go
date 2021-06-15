package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	nanoid "github.com/matoous/go-nanoid/v2"
	"github.com/rs/cors"

	"github.com/ltekieli/shorturl/cache"
	"github.com/ltekieli/shorturl/cache/memcache"
	"github.com/ltekieli/shorturl/db"
	"github.com/ltekieli/shorturl/log"
)

const RequestLimit = 2048

type Config struct {
	port      *uint
	dbIp      *string
	dbPort    *uint
	cacheIp   *string
	cachePort *uint
}

var (
	gConfig Config
	gCache  cache.Cache
	gDb     db.Database
	gServer *http.Server
)

type LongLink struct {
	Url string `json:"url"`
}

type ShortLink struct {
	Url string `json:"url"`
}

func shorten(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, RequestLimit))
	if err != nil {
		log.Errorf("Failed to read shorten request: %s", err)
		http.Error(w, "Failed to read shorten request", http.StatusBadRequest)
		return
	}

	var longLink LongLink
	err = json.Unmarshal(reqBody, &longLink)
	if err != nil {
		log.Errorf("Shorten request contains invalid LongLink: %s", err)
		http.Error(w, "Shorten request contains invalid LongLink", http.StatusBadRequest)
		return
	}

	_, err = url.ParseRequestURI(longLink.Url)
	if err != nil {
		log.Errorf("Invalid URI received: %s", err)
		http.Error(w, "Invalid URI received", http.StatusBadRequest)
		return
	}

	log.Infof("Shorten request received with: %s", longLink.Url)

	cached, ok := gCache.FetchByLong(longLink.Url)
	if ok {
		shortLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(shortLink)
	} else {
		sids, err := gDb.FetchByLong(longLink.Url)
		if err != nil {
			log.Errorf("Cannot fetch from database: %s", err)
			http.Error(w, "Cannot fetch from database", http.StatusInternalServerError)
			return
		}

		if len(sids) == 0 {
			sid, _ := nanoid.New()
			err := gDb.Insert(longLink.Url, sid)
			if err != nil {
				log.Errorf("Cannot insert to database: %s", err)
				http.Error(w, "Cannot insert to database", http.StatusInternalServerError)
				return
			}
			sids = append(sids, sid)
		} else if len(sids) > 1 {
			log.Error("Too many entries in the database for the same long link")
			http.Error(w, "Too many entries in the database for the same long link", http.StatusInternalServerError)
			return
		}

		gCache.Update(longLink.Url, sids[0])
		shortLink := ShortLink{Url: sids[0]}
		json.NewEncoder(w).Encode(shortLink)
	}
}

func resolve(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, RequestLimit))
	if err != nil {
		log.Errorf("Failed to read resolve request: %s", err)
		http.Error(w, "Failed to read resolve request", http.StatusBadRequest)
		return
	}

	var shortLink ShortLink
	err = json.Unmarshal(reqBody, &shortLink)
	if err != nil {
		log.Errorf("Resolve request contains invalid ShortLink: %s", err)
		http.Error(w, "Resolve request contains invalid ShortLink", http.StatusBadRequest)
		return
	}

	log.Infof("Resolve request received with: %s", shortLink.Url)

	cached, ok := gCache.FetchByShort(shortLink.Url)
	if ok {
		longLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(longLink)
	} else {
		lid, err := gDb.FetchByShort(shortLink.Url)
		if err != nil {
			log.Errorf("Cannot fetch from database: %s", err)
			http.Error(w, "Cannot fetch from database", http.StatusInternalServerError)
			return
		}

		if len(lid) == 0 {
			log.Error("Short link does not exist")
			http.Error(w, "Short link does not exist", http.StatusInternalServerError)
			return
		} else if len(lid) > 1 {
			log.Error("Too many entries in the database for the same short link")
			http.Error(w, "Too many entries in the database for the same short link", http.StatusInternalServerError)
			return
		}

		gCache.Update(lid[0], shortLink.Url)
		longLink := LongLink{Url: lid[0]}
		json.NewEncoder(w).Encode(longLink)
	}
}

func connectDatabase(ip string, port uint16) error {
	log.Info("Connecting to database...")
	uri := fmt.Sprintf("mongodb://%s:%d", ip, port)
	var err error
	gDb, err = db.Connect(uri, "shorturls", "shorturls")
	if err != nil {
		return err
	}
	log.Info("Successfully connected")
	return nil
}

func disconnectDatabase() {
	gDb.Disconnect()
}

func connectCache(ip string, port uint16) {
	log.Info("Setting up cache...")
	gCache = memcache.New(fmt.Sprintf("%s:%d", ip, port))
	log.Info("Successfully set up cache")
}

func startServer(ip string, port uint16) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/shorten", shorten).Methods("POST")
	router.HandleFunc("/api/resolve", resolve).Methods("POST")

	handler := cors.Default().Handler(router)

	gServer = &http.Server{Addr: fmt.Sprintf("%s:%d", ip, port), Handler: handler}

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
			case <-time.After(5 * time.Second):
				if err := gCache.Ping(); err != nil {
					log.Error("Cache not available")
				}
			}
		}
	}()
}

func getConfig() {
	gConfig = Config{}
	gConfig.port = flag.Uint("port", 8080, "Server port.")
	gConfig.dbIp = flag.String("db-ip", "", "Database IP address. (Required)")
	gConfig.dbPort = flag.Uint("db-port", 27017, "Database port.")
	gConfig.cacheIp = flag.String("cache-ip", "", "Cache IP address. (Required)")
	gConfig.cachePort = flag.Uint("cache-port", 11211, "Cache port.")

	flag.Parse()

	if *gConfig.dbIp == "" || *gConfig.cacheIp == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func main() {
	getConfig()
	log.Info("Starting URL shortener...")
	if err := connectDatabase("192.168.30.2", uint16(*gConfig.dbPort)); err != nil {
		panic(err)
	}
	defer disconnectDatabase()
	connectCache("192.168.30.3", uint16(*gConfig.cachePort))
	startServer("0.0.0.0", uint16(*gConfig.port))
	defer stopServer()
	waitForInterrupt()
}
