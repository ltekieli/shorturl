package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	nanoid "github.com/matoous/go-nanoid/v2"
	"io"
	"log"
	"net/http"
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
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var longLink LongLink
	err = json.Unmarshal(reqBody, &longLink)
	if err != nil {
		http.Error(w, "Invalid long link request", http.StatusInternalServerError)
		return
	}
	log.Print(longLink)

	cached, ok := MappingLongToShort[longLink.Url]
	if ok {
		shortLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(shortLink)
	} else {
		sid, err := db.FetchByLong(longLink.Url)
		if err != nil {
			sid, _ = nanoid.New()
			err := db.Insert(longLink.Url, sid)
			if err != nil {
				http.Error(w, "Cannot insert to database", http.StatusInternalServerError)
				return
			}
		}
		MappingLongToShort[longLink.Url] = sid
		MappingShortToLong[sid] = longLink.Url
		shortLink := ShortLink{Url: sid}
		json.NewEncoder(w).Encode(shortLink)
	}
}

func resolve(w http.ResponseWriter, r *http.Request) {
	reqBody, err := io.ReadAll(io.LimitReader(r.Body, RequestLimit))
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var shortLink ShortLink
	err = json.Unmarshal(reqBody, &shortLink)
	if err != nil {
		http.Error(w, "Invalid short link request", http.StatusInternalServerError)
		return
	}
	log.Print(shortLink)

	cached, ok := MappingShortToLong[shortLink.Url]
	if ok {
		longLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(longLink)
	} else {
		lid, err := db.FetchByShort(shortLink.Url)
		if err != nil {
			http.Error(w, "Short link does not exist", http.StatusInternalServerError)
		} else {
			MappingLongToShort[lid] = shortLink.Url
			MappingShortToLong[shortLink.Url] = lid
			longLink := LongLink{Url: lid}
			json.NewEncoder(w).Encode(longLink)
		}
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
