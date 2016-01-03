package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/f2prateek/hn2instapaper-server/Godeps/_workspace/src/github.com/f2prateek/go-instapaper"
	"github.com/f2prateek/hn2instapaper-server/Godeps/_workspace/src/github.com/f2prateek/hn2instapaper/hn"
	"github.com/f2prateek/hn2instapaper-server/Godeps/_workspace/src/github.com/f2prateek/semaphore"
	"github.com/f2prateek/hn2instapaper-server/Godeps/_workspace/src/github.com/gohttp/response"
	"github.com/f2prateek/hn2instapaper-server/Godeps/_workspace/src/github.com/tj/docopt"
)

const (
	usage = `hn2instapaper.

Usage:
  hn2instapaper [--addr=<a>]
  hn2instapaper -h | --help
  hn2instapaper --version

Options:
  -h --help      Show this screen.
  --version      Show version.
  --addr=<a>     Bind address [default: :8080].`
	version = "0.1.0"
)

var homeTemplate = loadTemplate("home")
var loginTemplate = loadTemplate("login")
var statsTemplate = loadTemplate("stats")

type Page struct {
	Name string
}

// Return the parsed template file at `templates/{name}.tmpl.html` by composing
// it with `templates/base.tmpl.html`.
func loadTemplate(name string) *template.Template {
	b := "templates/base.tmpl.html"
	t := "templates/" + name + ".tmpl.html"
	return template.Must(template.New(name).ParseFiles(b, t))
}

func home(w http.ResponseWriter, r *http.Request) {
	err := homeTemplate.ExecuteTemplate(w, "base", Page{"Home"})
	if err != nil {
		log.Println("error rendering template", err)
		response.InternalServerError(w)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	err := loginTemplate.ExecuteTemplate(w, "base", Page{"Login"})
	if err != nil {
		log.Println("error rendering template", err)
		response.InternalServerError(w)
	}
}

const DefaultLimit = 50

func parseLimit(r *http.Request) int {
	l := r.URL.Query().Get("limit")
	if l == "" {
		return DefaultLimit
	}

	i, err := strconv.Atoi(l)
	if err != nil {
		log.Println("could not parse limit", l)
		return DefaultLimit
	}

	return i
}

func importStories(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		log.Printf("%v method not allowed for /import", r.Method)
		response.MethodNotAllowed(w)
		return
	}

	username, password, ok := r.BasicAuth()
	if !ok {
		response.Forbidden(w)
		return
	}
	limit := parseLimit(r)
	log.Printf("importing %v stories for %v", limit, username)

	hnClient := hn.New()
	stories, err := hnClient.TopStories()
	if err != nil {
		response.BadGateway(w, err)
		return
	}

	var errs []error
	var addedStories []hn.Item
	instapaperClient := instapaper.New(username, password)
	s := semaphore.New(10)
	var wg sync.WaitGroup
	for i, id := range stories {
		if i >= limit {
			break
		}

		wg.Add(1)
		s.Acquire(1)

		go func(id int) {
			defer wg.Done()
			defer s.Release(1)

			story, err := hnClient.GetPost(id)
			if err != nil {
				errs = append(errs, err)
				return
			}
			if story.URL == nil {
				log.Printf("skipping story %v with title %v", id, story.Title)
				return
			}

			_, err = instapaperClient.Add(instapaper.AddParams{
				URL:   *story.URL,
				Title: story.Title,
			})
			if err != nil {
				errs = append(errs, err)
				log.Println("error adding", story, err)
				return
			}
			addedStories = append(addedStories, story)
		}(id)
	}
	wg.Wait()

	if len(errs) != 0 {
		response.BadGateway(w, errs, addedStories)
		return
	}

	log.Printf("imported %v stories for %v", len(addedStories), username)
	response.JSON(w, addedStories)
}

func auth(w http.ResponseWriter, r *http.Request) {
	username, password, ok := r.BasicAuth()
	if !ok {
		response.Forbidden(w)
		return
	}

	instapaperClient := instapaper.New(username, password)
	ok, err := instapaperClient.Authenticate()
	if err != nil {
		response.InternalServerError(w)
		return
	}

	if ok {
		response.OK(w)
		return
	}

	response.Forbidden(w)
}

func main() {
	args, err := docopt.Parse(usage, nil, true, version, false)
	if err != nil {
		log.Fatal(err)
	}

	addr := args["--addr"].(string)
	if addr == ":8080" {
		envPort := os.Getenv("PORT")
		if envPort != "" {
			log.Println("using $PORT", envPort)
			addr = ":" + envPort
		}
	}

	http.HandleFunc("/", home)
	http.HandleFunc("/login", login)
	http.HandleFunc("/import", importStories)
	http.HandleFunc("/auth", auth)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("starting hn2instapaper server on", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
