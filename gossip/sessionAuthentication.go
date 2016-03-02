package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"time"

	"github.com/nu7hatch/gouuid"
	"golang.org/x/crypto/bcrypt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//We should redo all this session stuff with JWT sometime soon
//SessionTimeout in terms of minutes.
var sessionTimeout = 30.0

type userInformation struct {
	Username        string
	Password        []byte
	Uuid            uuid.UUID
	MessageSequence int
}

type userInformationWithID struct {
	Username        string
	Password        []byte
	Uuid            uuid.UUID
	MessageSequence int
	ID              bson.ObjectId "_id"
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

func removeSession(sessionID string) {
	log.Println("Removing Session", sessionID)
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Sessions")

	c.Remove(bson.M{"sessionkey": sessionID})
}

func incrementMessageNumber(userID bson.ObjectId) {
	log.Println("Incrementing count for user ", userID, ".... ")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Users")

	c.UpdateId(userID, bson.M{"$inc": bson.M{"messagesequence": 1}})

	log.Println("Count Incremented. ")
}

func createSession(sessionID string, userID string) {
	log.Println("Creating Session...")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Sessions")

	toInsert := sessionInfo{sessionID, userID, time.Now(), time.Now()}
	c.Insert(&toInsert)
	log.Println("Done.")
}

func findUserBySession(sessionKey string) userInformationWithID {
	log.Println("Retrieving User for session ", sessionKey, "...")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Sessions")

	sessionInfo := sessionInfo{}
	_ = c.Find(bson.M{"sessionkey": sessionKey}).One(&sessionInfo)
	log.Println("Session Retrieved.")

	result := findUserByUserID(sessionInfo.UserID)

	return result
}

func findUserByUserID(userID string) userInformationWithID {
	log.Println("Retriving User Information for user ID ", userID, "...")
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Users")

	result := userInformationWithID{}
	_ = c.FindId(bson.ObjectIdHex(userID)).One(&result)

	log.Println(result)

	log.Println("Retrieved User Information for ", result.Username, ".")

	return result
}

func checkSessionByUsername(username string) sessionInfo {
	log.Println("Retrieving Session for ", username)
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Sessions")

	result := sessionInfo{}
	_ = c.Find(bson.M{"username": username}).One(&result)

	log.Println("Session Key: ", result.SessionKey)

	return result
}

//check to see if the session is active/valid.
func checkSession(sessionKey string) bool {

	log.Println("Checking Session...")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Sessions")

	result := sessionInfo{}
	_ = c.Find(bson.M{"sessionkey": sessionKey}).One(&result)

	if result.SessionKey != "" {
		log.Println("Session key exists.")
		elapsed := time.Since(result.LastSeen)
		if elapsed.Minutes() > sessionTimeout {
			log.Println("Session Inactive.")
			//if our session was too long ago.
			c.Remove(bson.M{"sessionkey": sessionKey})
			return false
		}
		log.Println("Session Active.")
		//There was a session key, and it hasn't expired yet.
		return true

	}
	log.Println("Session key:", sessionKey, " does not exist.")
	return false
}

func checkForUsername(username string) bool {
	log.Println("Checking name ", username, "...")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Users")

	result := userInformationWithID{}
	err = c.Find(bson.M{"username": username}).One(&result)
	if err == nil {
		log.Println("Name Exists")
		return false
	}
	log.Println("No Name Exists")
	return true

}

func checkCredentials(username string, password string) (userInformationWithID, bool) {
	log.Println("Check Credentials for: ", username)

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Users")

	result := userInformationWithID{}
	err = c.Find(bson.M{"username": username}).One(&result)
	check(err)

	log.Println("Results: \n ID:", result.ID)
	log.Println("Username: ", result.Username)

	if result.Password != nil {
		err = bcrypt.CompareHashAndPassword(result.Password, []byte(password))
		if err == nil {
			return result, true
		}
	}

	return result, false
}

func setLastSeen(sessionKey string) {
	log.Println("Setting last seen...")
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("Sessions")

	log.Println("Session key: ", sessionKey)
	c.Update(bson.M{"sessionkey": sessionKey}, bson.M{"$set": bson.M{"lastseen": time.Now()}})

	var s = sessionInfo{}

	c.Find(bson.M{"sessionkey": sessionKey}).One(&s)

	log.Println("New session Time: ", s.LastSeen)

	log.Println("Last seen set to now.")
}

func generateUUID(username string, password string) *uuid.UUID {
	toreturn, err := uuid.NewV5(uuid.NamespaceURL, []byte(username+password+sourceAddress))
	check(err)
	return toreturn
}

func register(username string, password string) {
	log.Println("Register")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	pass, err := bcrypt.GenerateFromPassword([]byte(password), 10)
	check(err)

	toSave := userInformation{username, pass, *generateUUID(username, string(pass)), 0}
	c := session.DB(dbName).C("Users")
	err = c.Insert(&toSave)

	check(err)
}
