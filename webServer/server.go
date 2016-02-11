package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
)

func takeAction(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Taking Action")
	fmt.Println("Checking Cookies")

	cookie, err := r.Cookie("Session")
	check(err)

	if checkSession(cookie.Value) {
		fmt.Fprintf(w, "Success")
	} else {
		fmt.Fprintf(w, "Failure")
	}
}

func getOathToken(userID string, ClientID string, ClientSecret string, RedirectURI string,
	userCode string) {
	_, err := http.Get(`https://foursquare.com/oauth2/access_token
		?client_id=` + ClientID + `
    &client_secret=` + ClientSecret + `
    &grant_type= NotSureWhatThissIs
    &redirect_uri=` + RedirectURI + `
    &code=` + userCode)

	check(err)
}
func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Testing")

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

		newCookie := http.Cookie{Name: "Session", Value: sessionID}
		http.SetCookie(w, &newCookie)
		fmt.Fprintf(w, "Success")
		return
	}
	fmt.Fprintf(w, "Failure")
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var port = flag.Int("port", 8080, "The port number you want the server running on. Default is 8080")

	flag.Parse()

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("Static/"))))
	http.HandleFunc("/api/login", login)
	http.HandleFunc("/api/action", takeAction)
	http.ListenAndServeTLS(":"+strconv.Itoa(*port), "server.pem", "server.key", nil)

}
