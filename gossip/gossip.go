package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
)

var dbName = "gossip"
var sourceAddress = "https://localhost"
var sourcePath = "/api/gossip"

//There must be a better way to do this, but whatever.
var PropagateRumorsChannel = make(chan RumorListToSend)

func takeAction(w http.ResponseWriter, r *http.Request) {
	log.Println("Taking Action")
	if checkCookie(r) {
		fmt.Fprintf(w, "Success")
	} else {
		fmt.Fprintf(w, "Failure")
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	//assume that the redirect is simply the /login endpoint.
	redirectAddress := r.URL.Host + "/login"
	log.Println(redirectAddress)

	http.Redirect(w, r, redirectAddress, 401)
}

func checkCookie(r *http.Request) bool {
	log.Println("Checking Cookies")

	cookie, err := r.Cookie("Session")
	if err != nil || !checkSession(cookie.Value) {
		return false
	}

	return true
}

func getData(w http.ResponseWriter, r *http.Request) {
	log.Println("Getting all Data")

	if !checkCookie(r) {
		fmt.Fprintf(w, "notLoggedIn")
		return
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	log.Println("Logging In")

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
		log.Println("Session created, generating cookie...")

		newCookie := http.Cookie{Name: "Session", Value: sessionID}
		http.SetCookie(w, &newCookie)

		log.Println("Done.")
		fmt.Fprintf(w, "Success")

		log.Println("Login operation succeeded")
		return
	}
	fmt.Fprintf(w, "Failure")
}

func sendMessageHandle(w http.ResponseWriter, r *http.Request) {
	if !checkCookie(r) {
		redirectToLogin(w, r)
		return
	}
	textByte, err := ioutil.ReadAll(r.Body)

	//the error here can't be that the cookie doesn't exist, since we checked that
	//in checkCookie())
	cookie, err := r.Cookie("Session")
	check(err)

	user := findUserBySession(cookie.Value)

	messageID := user.Uuid.String() + ":" + strconv.Itoa(user.MessageSequence)
	messageUUID := user.Uuid.String()
	messageNo := user.MessageSequence
	originator := user.Username
	text := string(textByte)

	toStore := Rumor{messageID, messageUUID, messageNo, originator, text}

	saveMessage(toStore)
	incrementMessageNumber(user.ID)
}

func addPeerHandle(w http.ResponseWriter, r *http.Request) {

}

func gossipHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("Received gossip message.")
	value, err := ioutil.ReadAll(r.Body)

	var wantMessage Want

	err = json.Unmarshal(value, &wantMessage)
	check(err)

	if err == nil {
		log.Println("Storing want message...")
		//It was a want message, handle appropriately.
		processWant(wantMessage, PropagateRumorsChannel)
		fmt.Fprintf(w, "Successfully stored want message")
		log.Println("Successfully stored want message.")
		return
	}

	var ReceivedRumor RumorMessage
	json.Unmarshal(value, wantMessage)

	check(err)
	if err == nil {
		log.Println("Storing Rumor...")
		information := strings.Split(":", ReceivedRumor.Rumor.MessageID)
		id, _ := strconv.Atoi(information[1])
		rumorToSave := Rumor{ReceivedRumor.Rumor.MessageID, information[0],
			id, ReceivedRumor.Rumor.Originator, ReceivedRumor.Rumor.Text}

		saveMessage(rumorToSave)

		fmt.Fprintf(w, "Successfuly stored rumor.")
		fmt.Println("Successfully stored rumor.")
		return
	}
	//fmt.Fprintf(w, "Error, unrecognized request. Requre either a want message or a rumor message.")
	http.Error(w, http.StatusText(400), 400)

}

func registerHandle(w http.ResponseWriter, r *http.Request) {
	log.Println("Registering")

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

	sourceAddress = sourceAddress + ":" + strconv.Itoa(*port)

	//We probably want to set the log path here.

	//load the configuration file with the known peers.

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("Static/"))))
	http.HandleFunc("/api/login", login)
	http.HandleFunc("/api/sendMessage", sendMessageHandle)
	http.HandleFunc("/api/register", registerHandle)
	http.HandleFunc("/api/addPeer", addPeerHandle)
	http.HandleFunc("/api/gossip", gossipHandle)

	//start our propagation thread.
	go PropagateRumors(PropagateRumorsChannel)
	//start our thread that will send out the "want" messages.
	go requestMessages()

	err := http.ListenAndServeTLS(":"+strconv.Itoa(*port), "server.pem", "server.key", nil)

	check(err)
}
