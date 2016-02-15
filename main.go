package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/gocrawl"
	"github.com/PuerkitoBio/goquery"

	"gopkg.in/amz.v1/aws"
	"gopkg.in/amz.v1/s3"
)

var (
	domains     []string
	destination string
	s3con       *s3.S3
	debug       bool
)

func printDebug(msg string) {
	if debug {
		fmt.Println(msg)
	}
}

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
	if err != nil {
		printDebug(fmt.Sprintf("Create File Error: %s\n", err.Error()))
	}

	defer f.Close()

	_, err = f.WriteString(str)
	if err != nil {
		printDebug(fmt.Sprintf("Write File Error: %s\n", err.Error()))
	}

	f.Sync()
}

func writeToS3(uri string, str string, ctype string) {
	uri = strings.TrimPrefix(uri, "s3://")
	parts := strings.SplitN(uri, "/", 2)
	path := parts[1]
	bucket := s3con.Bucket(parts[0])

	data := []byte(str)
	err := bucket.Put(path, data, ctype, s3.PublicRead)
	if err != nil {
		printDebug(fmt.Sprintf("S3 Put Error: %s\n", err.Error()))
	}
}

func writeFile(path string, str string, ctype string) {
	if strings.HasPrefix(destination, "s3://") {
		writeToS3(path, str, ctype)
	} else {
		writeToDisk(path, str)
	}
}

// gocrawl functions

type Ext struct {
	*gocrawl.DefaultExtender
}

func (this *Ext) Visit(ctx *gocrawl.URLContext, res *http.Response, doc *goquery.Document) (interface{}, bool) {
	// Use the goquery document or res.Body to manipulate the data

	printDebug(fmt.Sprintf("Visiting: %s\n", ctx.URL()))

	u, _ := url.Parse(fmt.Sprintf("%v", ctx.URL()))

	path := u.Path
	if u.Path == "" {
		path = "/index.html"
	} else if strings.HasSuffix(u.Path, "/") {
		path = u.Path + "index.html"
	}

	printDebug(fmt.Sprintf("Writing to: %s\n", path))

	contentType := res.Header.Get("Content-Type")
	writeFile(destination+path, strings.TrimPrefix(fmt.Sprintf("%v", res.Body), "{"), contentType)

	return nil, true
}

func (this *Ext) Filter(ctx *gocrawl.URLContext, isVisited bool) bool {
	if isVisited {
		return false
	}

	match, _ := regexp.MatchString("wp-admin", ctx.URL().Host)
	if match {
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
	maxArg := flag.String("max", "1000", "The maximum amount of pages to crawl on the site.")
	debugArg := flag.String("debug", "", "If set, prints debug statements.")
	flag.Parse()

	if *debugArg != "" {
		debug = true
	}

	if *domainsArg == "" {
		err := errors.New("Missing domain! Run with -h to see all options.\n")
		fmt.Print(err)
		os.Exit(2)
	}

	domains = strings.Split(fmt.Sprintf("%v", *domainsArg), ",")

	if *destArg == "" {
		err := errors.New("Missing destination! Run with -h to see all options.\n")
		fmt.Print(err)
		os.Exit(2)
	}

	destination = strings.TrimSuffix(fmt.Sprintf("%v", *destArg), "/")

	if strings.HasPrefix(destination, "s3://") {
		// The AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY environment variables are used.
		auth, err := aws.EnvAuth()
		if err != nil {
			panic(err.Error())
		}
		s3con = s3.New(auth, aws.USEast)
	} else {
		flag, _ := dirExists(destination)
		if flag == false {
			os.MkdirAll(destination, 0755)
		}
	}

	ext := &Ext{&gocrawl.DefaultExtender{}}

	// Custom options
	opts := gocrawl.NewOptions(ext)
	opts.CrawlDelay = 1 * time.Second
	opts.LogFlags = gocrawl.LogError
	opts.SameHostOnly = false
	opts.MaxVisits, _ = strconv.Atoi(*maxArg)

	c := gocrawl.NewCrawlerWithOptions(opts)
	printDebug(fmt.Sprintf("Starting crawler on %s\n", domains[0]))
	c.Run("http://" + domains[0])
}
