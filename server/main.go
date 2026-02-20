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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	apiResponse, _ := json.Marshal(&responseMap)
	fmt.Fprintf(w, string(apiResponse))
}

func returnLongResponse(w http.ResponseWriter, req *http.Request) {
	beeMovieScript := `According to all known laws of aviation, there is no way a bee 
should be able to fly. Its wings are too small to get 
its fat little body off the ground. The bee, of course, 
flies anyway because bees don't care what humans think 
is impossible. Yellow, black. Yellow, black. Yellow, black. 
Yellow, black. Ooh, black and yellow! Let's shake it up a little.
Barry! Breakfast is ready!
Coming!
Hang on a second.
Hello?
Barry?
Adam?
Can you believe this is happening?
I can't.
I'll pick you up.
Looking sharp.
Use the stairs, Your father paid good money for those.
Sorry. I'm excited.
Here's the graduate.
We're very proud of you, son.
A perfect report card, all B's.
Very proud.
Ma! I got a thing going here.
You got lint on your fuzz.
Ow! That's me!
Wave to us! We'll be in row 118,000.
Bye!
Barry, I told you, stop flying in the house!
Hey, Adam.
Hey, Barry.
Is that fuzz gel?
A little. Special day, graduation.
Never thought I'd make it.
Three days grade school, three days high school.
Those were awkward.
Three days college. I'm glad I took a day and hitchhiked around The Hive.
You did come back different.
Hi, Barry. Artie, growing a mustache? Looks good.
Hear about Frankie?
Yeah.
You going to the funeral?
No, I'm not going.
Everybody knows, sting someone, you die.
Don't waste it on a squirrel.
Such a hothead.
I guess he could have just gotten out of the way.
I love this incorporating an amusement park into our day.
That's why we don't need vacations.
Boy, quite a bit of pomp under the circumstances.
Well, Adam, today we are men.
We are!
Bee-men.
Amen!
Hallelujah!
Students, faculty, distinguished bees,
please welcome Dean Buzzwell.
Welcome, New Hive City graduating class of 9:15.
That concludes our ceremonies And begins your career at Honex Industries!
Will we pick our job today?
I heard it's just orientation.
Heads up! Here we go.
Keep your hands and antennas inside the tram at all times.
Wonder what it'll be like?
A little scary.
Welcome to Honex, a division of Honesco and a part of the Hexagon Group.
This is it!
Wow.
Wow.
We know that you, as a bee, have worked your whole life to get to the point where you can work for your whole life.
Honey begins when our valiant Pollen Jocks bring the nectar to The Hive.
Our top-secret formula is automatically color-corrected, scent-adjusted and bubble-contoured into this soothing sweet syrup with its distinctive golden glow you know as... Honey!
That girl was hot.
She's my cousin!
She is?
Yes, we're all cousins.
Right. You're right.
At Honex, we constantly strive to improve every aspect of bee existence.
These bees are stress-testing a new helmet technology.
What do you think he makes?
Not enough.
Here we have our latest advancement, the Krelman.
What does that do?
Catches that little strand of honey that hangs after you pour it.
Saves us millions.
Can anyone work on the Krelman?
Of course. Most bee jobs are small ones.
But bees know that every small job, if it's done well, means a lot.
But choose carefully because you'll stay in the job you pick for the rest of your life.
The same job the rest of your life? I didn't know that.
What's the difference?
You'll be happy to know that bees, as a species, haven't had one day off in 27 million years.
So you'll just work us to death?
We'll sure try.
Wow! That blew my mind!
"What's the difference?"
How can you say that?
One job forever?
That's an insane choice to have to make.
I'm relieved. Now we only have to make one decision in life.
But, Adam, how could they never have told us that?`
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, beeMovieScript)
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
	http.HandleFunc("/long-response", returnLongResponse)
	http.HandleFunc("/4xxtest", return404)
	http.HandleFunc("/5xxtest", return500)
	http.HandleFunc("/user/{key}", mockAPI)
	http.HandleFunc("/users", returnMockUsers)

	http.ListenAndServe(":8090", nil)
}
