package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"gopkg.in/mgo.v2"
)

type foursquareInformation struct {
	UserID    string
	UserToken string
	Tastes    []string
	Saved     []string
}

func takeAction(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Taking Action")
	if checkCookie(r) {
		fmt.Fprintf(w, "Success")
	} else {
		fmt.Fprintf(w, "Failure")
	}
}

func checkCookie(r *http.Request) bool {
	fmt.Println("Checking Cookies")

	cookie, err := r.Cookie("Session")
	if err != nil || !checkSession(cookie.Value) {
		return false
	}

	return true
}

func getOathTokenHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("In OathTokenHandler")

	if !checkCookie(r) {
		fmt.Fprintf(w, "InvalidSession")
		fmt.Println("InvalidSession")
		return
	}

	code := r.URL.Query().Get("code")
	fmt.Println("code: ", code)

	if code == "" {
		fmt.Fprintf(w, "Invalid Request")
		return
	}
	cookie, _ := r.Cookie("Session")

	fmt.Println("Going to go and get the OAuthToken")
	getOathToken(findUserBySession(cookie.Value).ID.String(), code)

	fmt.Fprintf(w, "Success")
}

func getOathToken(userID string, userCode string) {
	fmt.Println("Getting the OAuthToken")
	//CHANGE FROM LOCALHOST

	requestString := "https://foursquare.com/oauth2/access_token?client_id=5ATHFEOTK5EU23DGQXCJ4XHYF1OWTBDIIV2CHXBAYQN0X5IO&client_secret=F1SZ1YRLHF4RURVU40QTC5NCB4Y3AHPM4MMIXHFDCRZZD4R0&grant_type=authorization_code&redirect_uri=https://localhost/afterlanding.html&code=" + userCode
	fmt.Println(requestString)

	response, err := http.Get(requestString)
	if response.StatusCode != 200 {
		fmt.Println("error")
		body, _ := ioutil.ReadAll(response.Body)
		fmt.Println("Body: ", string(body))
		return
	}
	check(err)

	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("Body: ", string(body))

	strings := strings.SplitAfter(string(body), ":")
	fmt.Println("Strings after Split: ", strings)

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("test").C("foursqare")

	toInsert := foursquareInformation{userID, strings[1], make([]string, 0), make([]string, 0)}

	c.Insert(&toInsert)

	//I presume we would here go get the user's data
}

func getData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Getting all Data")

	if !checkCookie(r) {
		fmt.Fprintf(w, "notLoggedIn")
		return
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Logging In")

	decoder := json.NewDecoder(r.Body)

	var login loginInformation

	err := decoder.Decode(&login)
	check(err)

	User, value := checkCredentials(login.Username, login.Password)

	if value {
		//We need to add the sessionID to the DB and delete the old sessionKey
		//associated with that key, if there is one.

		//check if there is a current session associated with the user.
		session := checkSessionByUsername(User.ID.Hex())

		sessionID := generateSessionString(64)

		if session.SessionKey == "" {
			createSession(sessionID, User.ID.Hex())
		} else {
			removeSession(string(session.SessionKey))
			createSession(sessionID, User.ID.Hex())
		}
		fmt.Println("Session created, generating cookie...")

		newCookie := http.Cookie{Name: "Session", Value: sessionID}
		http.SetCookie(w, &newCookie)

		fmt.Println("Done.")
		fmt.Fprintf(w, "Success")

		fmt.Println("Login operation succeeded")
		return
	}
	fmt.Fprintf(w, "Failure")
}

func registerHandle(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Registering")

	decoder := json.NewDecoder(r.Body)

	var login loginInformation

	err := decoder.Decode(&login)
	check(err)

	//we should check to see if the username already exists
	if !checkForUsername(login.Username) {
		fmt.Fprintf(w, "Username Exists")
		return
	}
	register(login.Username, login.Password)

	fmt.Fprintf(w, "Success")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var port = flag.Int("port", 443, "The port number you want the server running on. Default is 8080")

	flag.Parse()

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("Static/"))))
	http.HandleFunc("/api/login", login)
	http.HandleFunc("/api/getAllData", getData)
	http.HandleFunc("/api/getOathToken", getOathTokenHandler)
	http.HandleFunc("/api/register", registerHandle)
	http.HandleFunc("/api/action", takeAction)
	err := http.ListenAndServeTLS(":"+strconv.Itoa(*port), "server.pem", "server.key", nil)

	check(err)

}
