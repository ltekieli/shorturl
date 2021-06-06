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
	}
	log.Print(longLink)

	cached, ok := MappingLongToShort[longLink.Url]
	if ok {
		shortLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(shortLink)
	} else {
		sid, _ := nanoid.New()
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
	}
	log.Print(shortLink)

	cached, ok := MappingShortToLong[shortLink.Url]
	if ok {
		longLink := ShortLink{Url: cached}
		json.NewEncoder(w).Encode(longLink)
	} else {
		http.Error(w, "Short link does not exist", http.StatusInternalServerError)
	}
}

func main() {
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
