package main

import (
	"encoding/json"
	"log"

	"github.com/playwright-community/playwright-go"
)

var (
	pw      *playwright.Playwright
	browser playwright.Browser
	page    playwright.Page
)

func init() {
	var err error
	pw, err = playwright.Run()
	if err != nil {
		log.Fatalf("Open playwright error: %v", err)
	}
	browser, err = pw.Chromium.Launch()
	if err != nil {
		log.Fatalf("Open browser error: %v", err)
	}
	page, err = browser.NewPage()
	if err != nil {
		log.Fatalf("Open page error: %v", err)
	}
}

type Article struct {
	Title string `json:"article"`
	Info  struct {
		PublishTime string `json:"publish-time"`
	} `json:"info"`
	Content string `json:"content"`
}

func main() {
	uris, err := uris("https://www.leisu.com/news/catalog-zuqiu")
	if err != nil {
		log.Printf("error: %v\n", err)
		return
	}
	for _, uri := range uris {
		if _, err = page.Goto(uri); err != nil {
			log.Printf("error: %v\n", err)
			continue
		}
		var article Article
		article.Title, _ = page.Locator(".article-detail .title").TextContent()
		article.Info.PublishTime, _ = page.Locator(".article-detail .article-info .publish-time").TextContent()
		var strict bool
		strict = true
		article.Content, err = page.InnerHTML(".article-detail .article-content", playwright.PageInnerHTMLOptions{
			Strict: &strict,
		})
		if article.Content != "" {
			raw, _ := json.Marshal(article)
			log.Printf("article: %s\n", raw)
		}
	}
}

func uris(p string) ([]string, error) {
	if _, err := page.Goto(p); err != nil {
		return nil, err
	}
	entries, err := page.Locator(".new-item").All()
	if err != nil {
		return nil, err
	}
	var uris []string
	for _, entry := range entries {
		uri, err := entry.GetAttribute("href")
		if err != nil {
			log.Printf("error: %v\n", err)
			continue
		}
		if uri != "" {
			uris = append(uris, uri)
		}
	}
	return uris, nil
}
