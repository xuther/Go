package main

import (
	"fmt"
  "net/http"
  "io/ioutil"
)

func loadIndexPage(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path[1:] == "" {
    fmt.Fprintf(w, getStaticPage("index.html"))
  } else {
    fmt.Fprintf(w, getStaticPage(r.URL.Path[1:]))
  }

}

func loadImages(w http.ResponseWriter, r *http.Request) {
  fmt.Println("images " + r.URL.Path[1:])
}

func getStaticPage(path string) string {
  path = "Static/" + path

  dat, err := ioutil.ReadFile(path)

  check(err)

  return string(dat)
}

func check(e error) {
    if e != nil {
        panic(e)
    }
}

func main() {
    http.HandleFunc("/", loadIndexPage)
    http.HandleFunc("/images/", loadImages)
    http.ListenAndServe(":8080", nil)
}
