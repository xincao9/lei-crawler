package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/playwright-community/playwright-go"
)

var (
	pw *playwright.Playwright
)

func init() {
	var err error
	pw, err = playwright.Run()
	if err != nil {
		log.Fatalf("Open playwright error: %v", err)
	}
}

func NewBrowser() (browser playwright.Browser, page playwright.Page, err error) {
	ips := dynamicIP()
	if ips == nil || len(ips) <= 0 {
		err = errors.New("dynamic ip acquisition failed")
		return
	}
	ip := ips[0]
	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Proxy: &playwright.Proxy{
			Server: fmt.Sprintf("http://%s", ip),
		},
	})
	if err != nil {
		return
	}
	page, err = browser.NewPage()
	if err != nil {
		return
	}
	return
}

type DynamicIP struct {
	Code int      `json:"code"`
	Data []string `json:"data"`
	Msg  string   `json:"msg"`
}

func dynamicIP() []string {
	resp, err := http.Get("http://list.sky-ip.net/user_get_ip_list?token=ywtPFyCzEaKqVjKG1666769437853&type=datacenter&qty=1&country=&time=5&format=json&protocol=http")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	buf := bytes.Buffer{}
	io.Copy(&buf, resp.Body)
	log.Printf("dynamicIP resp: %s", buf.String())
	var dip DynamicIP
	err = json.Unmarshal(buf.Bytes(), &dip)
	if err != nil {
		return nil
	}
	return dip.Data
}

type Article struct {
	Title string `json:"article"`
	Info  struct {
		PublishTime string `json:"publish-time"`
	} `json:"info"`
	Content string `json:"content"`
	Img     string `json:"img"`
}

func main() {
	browser, page, err := NewBrowser()
	defer func() {
		if browser != nil {
			err = browser.Close()
			log.Printf("browser.Close err: %v\n", err)
		}
		if pw != nil {
			err = pw.Stop()
			log.Printf("pw.Stop err: %v\n", err)
		}
	}()
	if err != nil {
		log.Printf("NewBrowser err: %v\n", err)
		return
	}
	uris, err := uris(page, "https://www.leisu.com/news/catalog-zuqiu")
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
		article.Content, _ = page.Locator(".article-detail .article-content").TextContent()
		article.Img, _ = page.Locator(".article-detail img").GetAttribute("src")
		if strings.TrimSpace(article.Content) != "" {
			raw, _ := json.Marshal(article)
			log.Printf("article: %s\n", raw)
		}
	}
}

func uris(page playwright.Page, p string) ([]string, error) {
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
