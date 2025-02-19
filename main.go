package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/playwright-community/playwright-go"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	pw *playwright.Playwright
	db *gorm.DB
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
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
	resp, err := http.Get("http://list.sky-ip.net/user_get_ip_list?token=xjOhpPWjBcQ3lIYH1739518234972&type=datacenter&qty=1&country=&time=5&format=json&protocol=http")
	if err != nil {
		return nil
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)
	buf := bytes.Buffer{}
	_, _ = io.Copy(&buf, resp.Body)
	log.Printf("dynamicIP resp: %s", buf.String())
	var dip DynamicIP
	err = json.Unmarshal(buf.Bytes(), &dip)
	if err != nil {
		return nil
	}
	return dip.Data
}

type Article struct {
	Id          int64     `json:"id" gorm:"primary_key"`
	Title       string    `json:"title" gorm:"column:title"`
	PublishTime string    `json:"publish-time" gorm:"column:publish_time"`
	Content     string    `json:"content" gorm:"column:content"`
	Img         string    `json:"img" gorm:"column:img"`
	Sport       string    `json:"sport" gorm:"column:sport"`
	Md5         string    `json:"md5" gorm:"column:md5"` // 根据Content计算MD5
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt   time.Time `json:"deleted_at" gorm:"column:deleted_at"`
}

func main() {
	t := flag.String("type", "", "competitions or crawArticle or install")
	flag.Parse()
	if *t == "competitions" {
		competitions()
	} else if *t == "crawArticle" {
		crawArticle() // 抓取全量文章
	} else if *t == "install" {
		install()
	}
}

func install() {
	err := playwright.Install()
	if err != nil {
		log.Fatalf("playwright.Install error: %v\n", err)
	} else {
		log.Printf("playwright.Install success\n")
	}
}

func competitions() {
	_, page, err := NewBrowser()
	if err != nil {
		log.Printf("NewBrowser err: %v\n", err)
		return
	}
	if _, err = page.Goto("https://www.leisu.com/data/zuqiu/comp-120"); err != nil {
		log.Printf("error: %v\n", err)
		return
	}
	m := make(map[string]string)
	competitions, _ := page.Locator(".competition-levels .competition a").All()
	for _, competition := range competitions {
		name, _ := competition.TextContent()
		id, _ := competition.GetAttribute("href")
		if id != "" {
			ss := strings.Split(id, "-")
			if len(ss) > 0 {
				id = ss[len(ss)-1]
			}
		}
		m[id] = name
	}
	data, _ := json.Marshal(m)
	log.Printf("%s", data)
}

func crawArticle() {
	var err error
	db, err = gorm.Open(mysql.Open("root:123456@tcp(localhost:3306)/leisu?charset=utf8&parseTime=true&loc=Local"))
	if err != nil {
		log.Fatalf("Open mysql error: %v\n", db)
	}
	configs := []struct {
		listUri   string
		startPage int
		endPage   int
		sport     string
	}{
		{
			listUri:   "https://www.leisu.com/news/catalog-zuqiu/%d", // 足球
			startPage: 1,
			endPage:   5462,
			sport:     "football",
		},
		{
			listUri:   "https://www.leisu.com/news/catalog-1/%d", // 足球综合
			startPage: 1,
			endPage:   2173,
			sport:     "football",
		},
		{
			listUri:   "https://www.leisu.com/news/catalog-lanqiu/%d", // 篮球
			startPage: 1,
			endPage:   1860,
			sport:     "basketball",
		},
		{
			listUri:   "https://www.leisu.com/news/catalog-4/%d", // 篮球综合
			startPage: 1,
			endPage:   152,
			sport:     "basketball",
		},
	}
	for _, config := range configs {
		for pageNo := config.startPage; pageNo <= config.endPage; pageNo++ {
			// 真正使用，需要改为并发模式
			crawler(fmt.Sprintf(config.listUri, pageNo), config.sport)
		}
	}
}

func crawler(listUri string, sport string) {
	_, page, err := NewBrowser()
	if err != nil {
		log.Printf("NewBrowser err: %v\n", err)
		return
	}
	uris, err := uris(page, listUri)
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
		article.Sport = sport
		article.Title, _ = page.Locator(".article-detail .title").TextContent()
		article.PublishTime, _ = page.Locator(".article-detail .article-info .publish-time").TextContent()
		article.Content, _ = page.Locator(".article-detail .article-content").TextContent()
		article.Img, _ = page.Locator(".article-detail img").GetAttribute("src")
		article.Content = strings.TrimSpace(article.Content)
		if article.Content != "" {
			article.Md5 = Sum(article.Content)
			save(&article)
		}
	}
}

func save(article *Article) {
	var oArticle Article
	db.First(&oArticle, "md5 = ?", article.Md5)
	if oArticle.Id > 0 {
		article.Id = oArticle.Id
	}
	db.Save(article)
}

func Sum(text string) string {
	harsher := md5.New()
	harsher.Write([]byte(text))
	hashBytes := harsher.Sum(nil)
	return hex.EncodeToString(hashBytes)
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
