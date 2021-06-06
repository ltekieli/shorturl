package main

import (
	"encoding/json"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"net/http"
)

type LongLink struct {
	Url string `json:"url"`
}

type ShortLink struct {
	Url string `json:"url"`
}

func shorten(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var longLink LongLink
	json.Unmarshal(reqBody, &longLink)
	log.Print(longLink)
	shortLink := ShortLink{Url: "35gdfgsde"}
	json.NewEncoder(w).Encode(shortLink)
}

func resolve(w http.ResponseWriter, r *http.Request) {
	reqBody, _ := ioutil.ReadAll(r.Body)
	var shortLink ShortLink
	json.Unmarshal(reqBody, &shortLink)
	log.Print(shortLink)
	longLink := ShortLink{Url: "http://page.com"}
	json.NewEncoder(w).Encode(longLink)
}

func main() {
	log.Printf("Starting URL shortener...")
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/api/shorten", shorten).Methods("POST")
	router.HandleFunc("/api/resolve", resolve).Methods("POST")
	if err := http.ListenAndServe(":8080", router); err != nil {
		log.Fatal(err)
	}
}
