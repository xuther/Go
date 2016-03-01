package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"strconv"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"time"
)

//Internal use, especially in DB. Make query easier.
type Rumor struct {
	MessageID   string
	MessageUUID string
	MessageNo   int
	Originator  string
	Text        string
}

//Format for communication over wires, interoperable
type RumorToSend struct {
	MessageID  string
	Originator string
	Text       string
}

type MessageList struct {
	Messagelist []Rumor
}

//Makes make creating JSON easy
type RumorMessage struct {
	Rumor    RumorToSend
	Endpoint string
}

//For compiling the messages that need to be sent to a given endpoint in
//response to a Want message
type RumorListToSend struct {
	Destination string
	Messages    []Rumor
}

type Peer struct {
	Endpoint string
	Want     []MessageTracking
}

type Want struct {
	Want     map[string]int
	EndPoint string
}

//this is the last from a given UUID that was sent to a peer.
type MessageTracking struct {
	MessageID   string
	MessageUUID string
	MessageNo   int
}

func saveMessage(rum Rumor) {
	log.Println("Saving Message from ", rum.Originator, "...")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("messages")

	c.UpsertId(rum.MessageUUID, bson.M{"$addToSet": bson.M{"messagelist": rum}})

	log.Println("Message Saved.")
}

func addWant(p Peer) {
	log.Println("Adding want complex for", p.Endpoint, "...")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("peers")

	c.Upsert(bson.M{"endpoint": p.Endpoint}, p)
}

//Function to check if a peer is in the system, and if not, add him.
func addPeer(address string) {
	log.Println("Establishing peer", address, "...")

	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("peers")

	//first we check to see if the peer already exists.
	var peer = Peer{}
	err = c.Find(bson.M{"address": address}).One(&peer)

	if err == nil {
		//the peer already exists, do nothing
		return
	}

	peer.Endpoint = address

	c.Insert(&peer)
	log.Println("Peer added.")
}

func evaluateNeededMessages(list []MessageTracking) []Rumor {
	log.Println("Evaluating needed messages...")
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("messages")

	var rumors []Rumor
	var tempRumors []Rumor
	var uuidList []string

	for _, mt := range list {
		c.Find(bson.M{"messageuuid": mt.MessageUUID, "messageno": bson.M{"$gt": mt.MessageNo}}).All(&tempRumors)
		uuidList = append(uuidList, mt.MessageUUID)
		rumors = append(rumors, tempRumors...)
	}

	c.Find(bson.M{"$not": bson.M{"messageuuid": uuidList}}).All(&tempRumors)
	rumors = append(rumors, tempRumors...)

	log.Println("Done. ", strconv.Itoa(len(rumors)), " messages found for forwarding.")

	return rumors
}

//Function to be run on it's own thread, will accept
func PropagateRumors(messagesToSend <-chan RumorListToSend) {

	for true {
		toSend := <-messagesToSend
		log.Println("Sending messages to ", toSend.Destination, "...")

		count := 0
		for _, msg := range toSend.Messages {
			count++
			sendRumor(msg, sourceAddress+sourcePath, toSend.Destination)
		}

		log.Println("Done sending messages. ", count, " messages sent.")
	}
}

func sendRumor(toSend Rumor, Source string, Dest string) {
	log.Println("Sending Message ", toSend.MessageID, "...")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	toSendRumor := RumorToSend{toSend.MessageUUID, toSend.Originator, toSend.Text}
	toSendFormat := RumorMessage{toSendRumor, Source}

	b, err := json.Marshal(toSendFormat)
	check(err)

	sendBuffer := bytes.NewBuffer(b)

	resp, err := client.Post(Dest, "application/json", sendBuffer)

	//log the error and move on
	if err != nil {
		log.Println(err)
	} else if resp.StatusCode != 200 {
		log.Println("Error on the send to ", Dest, ". Response Code ", resp.StatusCode)
	} else {
		log.Println("Done.")
	}
}

func processWant(wantMessage Want, messagestoSend chan<- RumorListToSend) {
	log.Println("Processing Want Message from ", wantMessage.EndPoint, "...")

	//translate to Peer object
	var translateBuffer []MessageTracking

	//Create what we need
	for k, v := range wantMessage.Want {
		toAppend := MessageTracking{k + ":" + strconv.Itoa(v), k, v}
		translateBuffer = append(translateBuffer, toAppend)
	}
	P := Peer{wantMessage.EndPoint, translateBuffer}

	//ensure that the peer exsits
	addPeer(wantMessage.EndPoint)

	//update our list of wants
	addWant(P)

	//We need to evaluate all the messages that the user doesn't have.
	messages := evaluateNeededMessages(translateBuffer)

	messageToSend := RumorListToSend{wantMessage.EndPoint, messages}
	if len(messageToSend.Messages) != 0 {
		log.Println("Queueing messages...")
		//Load the messages to send into the channel for a seperate thread to send
		messagestoSend <- messageToSend
		log.Println("Done.")
	} else {
		log.Println("No messages to be sent.")
	}

}

func sendWant(dest string, message Want) {
	log.Println("Sending want to ", dest, "...")
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr}

	b, err := json.Marshal(message)
	check(err)

	sendBuffer := bytes.NewBuffer(b)

	resp, err := client.Post(dest, "application/json", sendBuffer)

	//log the error and move on
	if err != nil {
		log.Println(err)
	} else if resp.StatusCode != 200 {
		log.Println("Error on the send to ", dest, ". Response Code ", resp.StatusCode)
	} else {
		log.Println("Done.")
	}
}

func buildWant(source string) Want {
	log.Println("Building the want message...")
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("messages")

	var AllMessageLists []MessageList

	var rumorList map[string]int
	rumorList = make(map[string]int)

	c.Find(nil).All(&AllMessageLists)

	var r Rumor
	for _, v := range AllMessageLists {
		c.FindId(v.Messagelist[0].MessageUUID).Sort("-MessageID").One(&r)
		rumorList[r.MessageUUID] = r.MessageNo
	}

	toReturn := Want{rumorList, source}
	log.Println("Want message built.")
	return toReturn
}

func getPeers() []string {
	log.Println("Getting all peers")
	session, err := mgo.Dial("localhost")
	check(err)
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB(dbName).C("peers")

	var peerEndpoints []string
	c.Find(nil).Select(bson.M{"_ID": 0, "endpoint": 1}).All(&peerEndpoints)
	log.Println("Done getting peers.")
	return peerEndpoints
}

//Function to periodically send out "want" messages - start on own thread.
func requestMessages() {
	for true {
		//Wait for 30 seconds
		time.Sleep(30 * time.Second)
		log.Println("Preparing to send want messages...")
		address := sourceAddress + sourcePath
		wantMessage := buildWant(address)

		//send want messages

		peers := getPeers()

		for _, p := range peers {
			sendWant(p, wantMessage)
		}
		if len(peers) != 0 {
			log.Println("Want messages sent.")
		} else {
			log.Println("No peers.")
		}
	}
}
