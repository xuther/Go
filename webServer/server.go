package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

func loadBasePage(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path[1:] == "" {
		fmt.Fprintf(w, getStaticPage("index.html"))
	} else {
		fmt.Fprintf(w, getStaticPage(r.URL.Path[1:]))
	}
}

func loadHTMLPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, getStaticPage(r.URL.Path[1:]))
}

func loadImages(w http.ResponseWriter, r *http.Request) {
	fmt.Println("images " + r.URL.Path[1:])
	fmt.Fprintf(w, getPage("images/"+r.URL.Path[1:]))
}

func getStaticPage(path string) string {
	return getPage("Static/" + path)
}

func getPage(path string) string {
	dat, err := ioutil.ReadFile(path)

	check(err)

	return string(dat)
}

func redirect(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query()

	switch path.Get("p") {
	case "1":
		http.Redirect(w, r, "http://www.google.com", 301)
		break
	case "2":
		http.Redirect(w, r, "http://www.byu.edu", 301)
		break
	case "3":
		http.Redirect(w, r, "http://www.amazon.com", 301)
		break
	}
}

func print(w http.ResponseWriter, r *http.Request) {
	toReturn := "Headers: \n"

	for k, v := range r.Header {
		toReturn += k + ": "
		for _, val := range v {
			toReturn += val
		}
		toReturn += "\n"
	}

	toReturn += "\nQuery: \n" + r.URL.RawQuery + "\n"

	toReturn += "Body: \n"

	body, _ := ioutil.ReadAll(r.Body)

	toReturn += string(body)

	fmt.Fprintf(w, toReturn)
}

func version(w http.ResponseWriter, r *http.Request) {
	t := r.Header.Get("Accept")
	w.Header().Set("Content-Type", "application/json")
	var toReturn JSONVersion

	switch t {
	case "application/vnd.byu.cs462.v1+json":
		toReturn = JSONVersion{"v1"}
		break
	case "application/vnd.byu.cs462.v2+json":
		toReturn = JSONVersion{"v2"}
		break
	}

	bytes, _ := json.Marshal(toReturn)

	fmt.Fprintf(w, string(bytes))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

//JSONVersion type to return the JSON version
type JSONVersion struct {
	Version string
}

func main() {
	var port = flag.Int("port", 8080, "The port number you want the server running on. Default is 8080")

	flag.Parse()

	fmt.Println(port)

	http.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir("Static/"))))
	http.HandleFunc("/pages/", loadHTMLPage)
	http.HandleFunc("/redirect/", redirect)
	http.HandleFunc("/print/", print)
	http.HandleFunc("/version/", version)
	http.ListenAndServe(":"+strconv.Itoa(*port), nil)
}
