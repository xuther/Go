package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//We should redo all this session stuff with JWT sometime soon
//SessionTimeout in terms of minutes.
var sessionTimeout = 10.0

type userInformation struct {
	Username string
	Password []byte
}

type userInformationWithID struct {
	Username string
	Password []byte
	ID       bson.ObjectId "_id"
}

type loginInformation struct {
	Username string
	Password string
}

type sessionInfo struct {
	SessionKey string
	UserID     string
	LoginTime  time.Time
	LastSeen   time.Time
}

func generateSessionString(s int) string {
	b := make([]byte, s)
	_, err := rand.Read(b)

	check(err)

	return base64.URLEncoding.EncodeToString(b)
}

func checkCredentials(username string, password string) (userInformationWithID, bool) {
	fmt.Println("Check Credentials")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("test").C("Users")

	result := userInformationWithID{}
	err = c.Find(bson.M{"username": username}).One(&result)
	check(err)

	fmt.Println("Results: \n ID:", result.ID)
	fmt.Println("Username: ", result.Username)

	if result.Password != nil {
		err = bcrypt.CompareHashAndPassword(result.Password, []byte(password))
		if err == nil {
			return result, true
		}
	}

	return result, false
}

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

func removeSession(sessionID string) {
	fmt.Println("Removing Session", sessionID)
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("test").C("Sessions")

	c.Remove(bson.M{"sessionkey": sessionID})

}

func createSession(sessionID string, userID string) {
	fmt.Println("Creating Session")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("test").C("Sessions")

	toInsert := sessionInfo{sessionID, userID, time.Now(), time.Now()}
	c.Insert(&toInsert)

}

func checkSessionByUsername(username string) sessionInfo {
	fmt.Println("Retrieving Session for ", username)
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("test").C("Sessions")

	result := sessionInfo{}
	_ = c.Find(bson.M{"userid": username}).One(&result)

	fmt.Println("Session Key: ", result.SessionKey)

	return result
}

//check to see if the session is active/valid.
func checkSession(sessionKey string) bool {

	fmt.Println("Checking Session")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("test").C("Sessions")

	result := sessionInfo{}
	_ = c.Find(bson.M{"sessionkey": sessionKey}).One(&result)

	fmt.Println("Results", result)

	if result.SessionKey != "" {
		elapsed := time.Since(result.LastSeen)
		if elapsed.Minutes() > sessionTimeout {
			return false
		}
		//There was a session key, and it hasn't expired yet.
		return true

	}
	return false
}

func register(username string, password string) {
	fmt.Println("Register")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	pass, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	check(err)

	toSave := userInformation{username, pass}
	c := session.DB("test").C("Users")
	err = c.Insert(&toSave)

	check(err)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var port = flag.Int("port", 8080, "The port number you want the server running on. Default is 8080")
	fmt.Println(checkCredentials("test", "test"))

	flag.Parse()

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("Static/"))))
	http.HandleFunc("/api/login", login)
	http.HandleFunc("/api/action", takeAction)
	http.ListenAndServe(":"+strconv.Itoa(*port), nil)

}
