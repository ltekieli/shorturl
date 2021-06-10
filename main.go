package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	nanoid "github.com/matoous/go-nanoid/v2"
	"io"
	"log"
	"net/http"
	"net/url"
)

const RequestLimit = 2048

var MappingLongToShort map[string]string
var MappingShortToLong map[string]string

var db Database

type LongLink struct {
	Url string `json:"url"`
}

type ShortLink struct {
	Url string `json:"url"`
}

func shorten(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, RequestLimit))
	if err != nil {
		log.Print("Failed to read shorten request")
		http.Error(w, "Failed to read shorten request", http.StatusBadRequest)
		return
	}

	var longLink LongLink
	err = json.Unmarshal(reqBody, &longLink)
	if err != nil {
		log.Print("Shorten request contains invalid LongLink")
		http.Error(w, "Shorten request contains invalid LongLink", http.StatusBadRequest)
		return
	}

	_, err = url.ParseRequestURI(longLink.Url)
	if err != nil {
		log.Print("Invalid URI received")
		http.Error(w, "Invalid URI received", http.StatusBadRequest)
		return
	}

	log.Printf("Shorten request received with: %s", longLink.Url)

	cached, ok := MappingLongToShort[longLink.Url]
	if ok {
		shortLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(shortLink)
	} else {
		sids, err := db.FetchByLong(longLink.Url)
		if err != nil {
			log.Print("Cannot fetch from database")
			http.Error(w, "Cannot fetch from database", http.StatusInternalServerError)
			return
		}

		if len(sids) == 0 {
			sid, _ := nanoid.New()
			err := db.Insert(longLink.Url, sid)
			if err != nil {
				log.Print("Cannot insert to database")
				http.Error(w, "Cannot insert to database", http.StatusInternalServerError)
				return
			}
			sids = append(sids, sid)
		} else if len(sids) > 1 {
			log.Print("Too many entries in the database for the same long link")
			http.Error(w, "Too many entries in the database for the same long link", http.StatusInternalServerError)
			return
		}

		MappingLongToShort[longLink.Url] = sids[0]
		MappingShortToLong[sids[0]] = longLink.Url
		shortLink := ShortLink{Url: sids[0]}
		json.NewEncoder(w).Encode(shortLink)
	}
}

func resolve(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, RequestLimit))
	if err != nil {
		log.Print("Failed to read resolve request")
		http.Error(w, "Failed to read resolve request", http.StatusBadRequest)
		return
	}

	var shortLink ShortLink
	err = json.Unmarshal(reqBody, &shortLink)
	if err != nil {
		log.Print("Resolve request contains invalid ShortLink")
		http.Error(w, "Resolve request contains invalid ShortLink", http.StatusBadRequest)
		return
	}

	log.Printf("Resolve request received with: %s", shortLink.Url)

	cached, ok := MappingShortToLong[shortLink.Url]
	if ok {
		longLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(longLink)
	} else {
		lid, err := db.FetchByShort(shortLink.Url)
		if err != nil {
			log.Print("Cannot fetch from database")
			http.Error(w, "Cannot fetch from database", http.StatusInternalServerError)
			return
		}

		if len(lid) == 0 {
			log.Print("Short link does not exist")
			http.Error(w, "Short link does not exist", http.StatusInternalServerError)
			return
		} else if len(lid) > 1 {
			log.Print("Too many entries in the database for the same short link")
			http.Error(w, "Too many entries in the database for the same short link", http.StatusInternalServerError)
		}

		MappingLongToShort[lid[0]] = shortLink.Url
		MappingShortToLong[shortLink.Url] = lid[0]
		longLink := LongLink{Url: lid[0]}
		json.NewEncoder(w).Encode(longLink)
	}
}

func main() {
	log.Print("Connecting to database...")

	uri := "mongodb://192.168.30.2:27017"
	var err error
	db, err = Connect(uri, "shorturls", "shorturls")
	if err != nil {
		panic(err)
	}
	defer db.Disconnect()
	log.Print("Successfully connected")

	log.Printf("Starting URL shortener...")
	MappingLongToShort = make(map[string]string)
	MappingShortToLong = make(map[string]string)
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/shorten", shorten).Methods("POST")
	router.HandleFunc("/api/resolve", resolve).Methods("POST")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
