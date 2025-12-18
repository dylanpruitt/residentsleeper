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

func mockAPI(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	apiVersion := req.Header.Get("Accept")
	if apiVersion == "application/json;v=1" {
		data := getMockAPIData(req.PathValue("key"), 1)
		if data != nil {
			w.WriteHeader(http.StatusOK)
			apiResponse, _ := json.Marshal(&data)
			fmt.Fprintf(w, string(apiResponse))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else if apiVersion == "application/json;v=2" {
		data := getMockAPIData(req.PathValue("key"), 2)
		if data != nil {
			w.WriteHeader(http.StatusOK)
			fmt.Println(data)
			apiResponse, _ := json.Marshal(&data)
			fmt.Println(string(apiResponse))
			fmt.Fprintf(w, string(apiResponse))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	} else {
		w.WriteHeader(http.StatusBadRequest)
		apiResponse, _ := json.Marshal("API version not supported. Must pass Accept header set to application/json;v=1 OR application/json;v=2")
		fmt.Fprintf(w, string(apiResponse))
	}
}

type V1Data struct {
	Name    string    `json:"name"`
	Account V1Account `json:"account"`
}

type V1Account struct {
	Money int `json:"money"`
}

type V2Data struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`

	Money int `json:"money"`
}

func getMockAPIData(key string, apiVersion int) interface{} {
	if key == "1" {
		if apiVersion == 1 {
			return V1Data{
				Name: "John Wick",
				Account: V1Account{
					Money: 100000,
				},
			}
		} else {
			return V2Data{
				FirstName: "John",
				LastName:  "Wick",
				Money:     100000,
			}
		}
	} else {
		return nil
	}
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
	http.HandleFunc("/user/{key}", mockAPI)

	http.ListenAndServe(":8090", nil)
}
