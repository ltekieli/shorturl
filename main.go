package main

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/gorilla/mux"
	nanoid "github.com/matoous/go-nanoid/v2"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ltekieli/shorturl/cache"
	"github.com/ltekieli/shorturl/db"
	"github.com/ltekieli/shorturl/log"
)

const RequestLimit = 2048

var (
	gCache cache.Cache
	gDb    db.Database
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
		log.Error("Failed to read shorten request")
		http.Error(w, "Failed to read shorten request", http.StatusBadRequest)
		return
	}

	var longLink LongLink
	err = json.Unmarshal(reqBody, &longLink)
	if err != nil {
		log.Error("Shorten request contains invalid LongLink")
		http.Error(w, "Shorten request contains invalid LongLink", http.StatusBadRequest)
		return
	}

	_, err = url.ParseRequestURI(longLink.Url)
	if err != nil {
		log.Error("Invalid URI received")
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
			log.Error("Cannot fetch from database")
			http.Error(w, "Cannot fetch from database", http.StatusInternalServerError)
			return
		}

		if len(sids) == 0 {
			sid, _ := nanoid.New()
			err := gDb.Insert(longLink.Url, sid)
			if err != nil {
				log.Error("Cannot insert to database")
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
		log.Error("Failed to read resolve request")
		http.Error(w, "Failed to read resolve request", http.StatusBadRequest)
		return
	}

	var shortLink ShortLink
	err = json.Unmarshal(reqBody, &shortLink)
	if err != nil {
		log.Error("Resolve request contains invalid ShortLink")
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
			log.Error("Cannot fetch from database")
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

func main() {
	log.Info("Starting URL shortener...")

	log.Info("Connecting to database...")
	uri := "mongodb://192.168.30.2:27017"
	var err error
	gDb, err = db.Connect(uri, "shorturls", "shorturls")
	if err != nil {
		panic(err)
	}
	defer gDb.Disconnect()
	log.Info("Successfully connected")

	log.Info("Setting up cache...")
	gCache = cache.New()
	log.Info("Successfully set up cache")

	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/shorten", shorten).Methods("POST")
	router.HandleFunc("/api/resolve", resolve).Methods("POST")
	server := &http.Server{Addr: ":8080", Handler: router}

	go func() {
		log.Info("Serving requests")
		if err := server.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				log.Info("Server shutdown")
			} else {
				log.Errorf("Server error during runtime: %s", err)
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Errorf("Server error during shutdown: %s", err)
	}
}
