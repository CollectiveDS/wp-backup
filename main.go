package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/gocrawl"
	"github.com/PuerkitoBio/goquery"
	//"launchpad.net/goamz/aws"
	//"launchpad.net/goamz/ec2"
)

var domains []string
var destination string

func dirExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func checkErr(e error) {
	if e != nil {
		panic(e)
	}
}

func writeToDisk(path string, str string) {
	// Create the directory
	parts := strings.Split(path, "/")
	parts = parts[0 : len(parts)-1]
	os.MkdirAll(strings.Join(parts, "/"), 0755)

	// Write the file
	f, err := os.Create(path)
	checkErr(err)

	defer f.Close()

	_, err = f.WriteString(str)
	checkErr(err)

	f.Sync()
}

// gocrawl functions

type Ext struct {
	*gocrawl.DefaultExtender
}

func (this *Ext) Visit(ctx *gocrawl.URLContext, res *http.Response, doc *goquery.Document) (interface{}, bool) {
	// Use the goquery document or res.Body to manipulate the data

	fmt.Printf("Visit: %s\n", ctx.URL())

	u, _ := url.Parse(fmt.Sprintf("%v", ctx.URL()))

	path := u.Path
	if u.Path == "" {
		path = "/index.html"
	} else if strings.HasSuffix(u.Path, "/") {
		path = u.Path + "index.html"
	}

	fmt.Printf("Writing to: %s\n", path)
	writeToDisk(destination+path, strings.TrimPrefix(fmt.Sprintf("%v", res.Body), "{"))

	return nil, true
}

func (this *Ext) Filter(ctx *gocrawl.URLContext, isVisited bool) bool {
	if isVisited {
		return false
	}

	for _, b := range domains {
		if b == ctx.URL().Host {
			return true
		}
	}

	return false
}

func main() {
	domainsArg := flag.String("domains", "", "The domain of the Wordpress site to archive.")
	destArg := flag.String("dest", "", "Destination local directory or S3 bucket.")
	flag.Parse()

	if *domainsArg == "" {
		err := errors.New("Missing domain!\n")
		fmt.Print(err)
		os.Exit(2)
	}

	domains = strings.Split(fmt.Sprintf("%v", *domainsArg), ",")

	if *destArg == "" {
		err := errors.New("Missing destination!\n")
		fmt.Print(err)
		os.Exit(2)
	}

	destination = strings.TrimSuffix(fmt.Sprintf("%v", *destArg), "/")

	flag, _ := dirExists(destination)
	if flag == false {
		os.MkdirAll(destination, 0755)
	}

	ext := &Ext{&gocrawl.DefaultExtender{}}

	// Custom options
	opts := gocrawl.NewOptions(ext)
	opts.CrawlDelay = 1 * time.Second
	opts.LogFlags = gocrawl.LogError
	opts.SameHostOnly = false
	opts.MaxVisits = 5

	c := gocrawl.NewCrawlerWithOptions(opts)
	fmt.Printf("Starting crawler on %s\n", domains[0])
	c.Run("http://" + domains[0])
}
