package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

var (
	gistToken   = os.Getenv("GIST_TOKEN")
	githubToken = os.Getenv("GITHUB_TOKEN")
	method      string
	url         string
)

// Flags ...
type Flags struct {
	token       string
	description string
	filename    string
	patch       string
	public      bool
}

// Response ...
type Response struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	ForksURL  string    `json:"forks_url"`
	HTMLURL   string    `json:"html_url"`
	CreatedAt time.Time `json:"created_at"`
}

// PostData ...
type PostData struct {
	Description string          `json:"description"`
	Public      bool            `json:"public"`
	Files       map[string]File `json:"files"`
}

// File ...
type File struct {
	Content string `json:"content"`
}

func getToken(token string) string {
	if token != "" {
		return token
	} else if gistToken != "" {
		return gistToken
	} else if githubToken != "" {
		return githubToken
	}
	return ""
}

func main() {
	flags := Flags{}
	flag.StringVar(&flags.token, "token", "", "token (or use GIST_TOKEN or GITHUB_TOKEN environment variables)")
	flag.StringVar(&flags.description, "description", "", "description of the gist")
	flag.StringVar(&flags.patch, "patch", "", "patch existing gist")
	flag.StringVar(&flags.filename, "filename", "", "filename for content from stdin")
	flag.BoolVar(&flags.public, "public", false, "make public gist")
	flag.Parse()

	token := getToken(flags.token)
	if token == "" {
		log.Fatalf("no token")
	}

	pd := PostData{}
	pd.Description = flags.description
	pd.Public = flags.public
	pd.Files = map[string]File{}
	if len(flag.Args()) == 0 {
		input, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("error reading input: %v", err)
		}
		f := File{}
		f.Content = string(input)
		filename := flags.filename
		if filename == "" {
			filename = "gist.txt"
		}
		pd.Files[filename] = f
	} else {
		for _, file := range flag.Args() {
			input, err := ioutil.ReadFile(file)
			if err != nil {
				log.Fatalf("error reading %s: %v", file, err)
			}
			f := File{}
			f.Content = string(input)
			pd.Files[path.Base(file)] = f
		}
	}

	bts, err := json.Marshal(pd)
	if err != nil {
		log.Fatalf("could not encode to json: %v", err)
	}

	if flags.patch != "" {
		method, url = "PATCH", "https://api.github.com/gists/"+flags.patch
	} else {
		method, url = "POST", "https://api.github.com/gists"
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(bts))
	if err != nil {
		log.Fatalf("could not create request: %v", err)
	}
	req.Header.Add("Authorization", "token "+token)
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")

	client := http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("could not make request: %v", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("could not read response: %v", err)
	}
	respdata := Response{}
	json.Unmarshal(body, &respdata)

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		fmt.Printf("Gist might have not been created, response status code: %d\n", resp.StatusCode)
		fmt.Printf("%s\n", body)
	} else {
		fmt.Printf("  ID:    %s\n", respdata.ID)
		fmt.Printf("  HTML:  %s\n", respdata.HTMLURL)
		fmt.Printf("  Date:  %s\n", respdata.CreatedAt)
	}
}
