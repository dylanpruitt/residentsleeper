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

	headerResponse, _ := json.Marshal(headerMap)
	fmt.Fprintf(w, string(headerResponse))
}

type PersonData struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Money     int    `json:"money"`
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

var mockAPIData = map[string]PersonData{
	"1": {FirstName: "John", LastName: "Wick", Money: 100000},
}

func mockAPI(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if req.Method == "GET" {
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
				apiResponse, _ := json.Marshal(&data)
				fmt.Fprintf(w, string(apiResponse))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		} else {
			w.WriteHeader(http.StatusBadRequest)
			apiResponse, _ := json.Marshal("API version not supported. Must pass Accept header set to application/json;v=1 OR application/json;v=2")
			fmt.Fprintf(w, string(apiResponse))
		}
	} else if req.Method == "POST" {
		data := PersonData{}
		err := json.NewDecoder(req.Body).Decode(&data)
		if err != nil {
			fmt.Println(err.Error())
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintf(w, err.Error())
			return
		}

		setMockAPIData(req.PathValue("key"), data)
		w.WriteHeader(http.StatusCreated)
		apiResponse, _ := json.Marshal(data)
		fmt.Fprintf(w, string(apiResponse))
	}
}

func getMockAPIData(key string, apiVersion int) interface{} {
	rawData, ok := mockAPIData[key]
	if ok {
		if apiVersion == 1 {
			return V1Data{
				Name: fmt.Sprintf("%s %s", rawData.FirstName, rawData.LastName),
				Account: V1Account{
					Money: rawData.Money,
				},
			}
		} else {
			return V2Data{
				FirstName: rawData.FirstName,
				LastName:  rawData.LastName,
				Money:     rawData.Money,
			}
		}
	}

	return nil
}

func setMockAPIData(key string, data PersonData) {
	mockAPIData[key] = data
}

func returnMockUsers(w http.ResponseWriter, req *http.Request) {
	responseMap := map[string]string{}
	for key, person := range mockAPIData {
		responseMap[key] = fmt.Sprintf("%s %s - $%d", person.FirstName, person.LastName, person.Money)
	}

	w.WriteHeader(http.StatusOK)
	apiResponse, _ := json.Marshal(&responseMap)
	fmt.Fprintf(w, string(apiResponse))
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
	http.HandleFunc("/users", returnMockUsers)

	http.ListenAndServe(":8090", nil)
}
