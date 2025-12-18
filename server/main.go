package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func hello(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	helloResponse, _ := json.Marshal("hello")
	fmt.Fprintf(w, string(helloResponse))
}

func headers(w http.ResponseWriter, req *http.Request) {
	headerMap := map[string]string{}
	for name, headers := range req.Header {
		for _, h := range headers {
			headerMap[name] = h
		}
	}

	w.Header().Set("Content-Type", "application/json")
	headerResponse, _ := json.Marshal(headerMap)
	fmt.Fprintf(w, string(headerResponse))
}

func return404(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusNotFound)
}

func return500(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusInternalServerError)
}

func main() {

	http.HandleFunc("/hello", hello)
	http.HandleFunc("/headers", headers)
	http.HandleFunc("/4xxtest", return404)
	http.HandleFunc("/5xxtest", return500)

	http.ListenAndServe(":8090", nil)
}
