package main

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup

type SitemapIndex struct {
	Locations []string `xml:"sitemap>loc"`
}

type News struct {
	Titles    []string `xml:"url>news>title"`
	Keywords  []string `xml:"url>news>keywords"`
	Locations []string `xml:"url>loc"`
}

type NewsMap struct {
	Keywords string
	Location string
}

type NewAggPage struct {
	Title string
	News  map[string]NewsMap
}

func newRoutine(c chan News, location string) {
	defer wg.Done()
	var n News
	location = strings.Trim(location, "\n")
	resp, err := http.Get(location)
	if err != nil {
		fmt.Println(err)
	}
	bytes, _ := ioutil.ReadAll(resp.Body)
	xml.Unmarshal(bytes, &n)
	resp.Body.Close()
	c <- n
}

func newsAggHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var s SitemapIndex
	news_map := make(map[string]NewsMap)
	resp, err := http.Get("https://www.washingtonpost.com/news-sitemaps/index.xml")
	if err != nil {
		fmt.Println(err)
	}
	bytes, _ := ioutil.ReadAll(resp.Body)
	xml.Unmarshal(bytes, &s)
	resp.Body.Close()

	queue := make(chan News, 30)

	for _, Location := range s.Locations {
		wg.Add(1)
		go newRoutine(queue, Location)
	}

	wg.Wait()
	close(queue)

	for elem := range queue {
		for idx, _ := range elem.Keywords {
			news_map[elem.Titles[idx]] = NewsMap{elem.Keywords[idx], elem.Locations[idx]}
		}
	}

	p := NewAggPage{Title: "", News: news_map}
	t, _ := template.ParseFiles("basic.html")
	fmt.Println(t.Execute(w, p))

	elapsed := time.Since(start)
	fmt.Printf("\ngo server without gorutine took %s\n", elapsed)
}

func main() {
	http.HandleFunc("/agg", newsAggHandler)
	http.ListenAndServe(":8000", nil)
}
