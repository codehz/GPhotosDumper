package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sync"

	"github.com/schollz/progressbar"
)

var re = regexp.MustCompile(`hash: '1', data:function\(\){return (\[(?:.|\n)+?\])\n}}\);`)

type meta []interface{}

func fetch(src string) (result meta, url url.URL) {
	resp, err := http.Get(src)
	if err != nil {
		panic(err)
	}
	url = *resp.Request.URL
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	sub := re.FindSubmatch(body)
	if len(sub) != 2 {
		panic("Unexpected document")
	}
	result = meta{}
	err = json.Unmarshal(sub[1], &result)
	if err != nil {
		panic(err)
	}
	return result, url
}

func listAlbum(src string) (result []string, title string) {
	x, url := fetch(src)
	if x[0] != nil {
		panic("Not a album!")
	}
	list := x[1].([]interface{})
	result = make([]string, len(list))
	for i, item := range list {
		content := item.([]interface{})
		target := content[0].(string)
		xurl := url
		xurl.Path = fmt.Sprintf("%s/photo/%s", url.Path, target)
		result[i] = xurl.String()
	}
	titlearr := x[3].([]interface{})
	title = titlearr[1].(string)
	return result, title
}

func sniffer(src string) (result string) {
	for {
		x, _ := fetch(src)
		result, ok := x[1].(string)
		if ok {
			return result
		}
		if !*parallel {
			fmt.Fprintln(os.Stderr, "retry...")
		}
	}
}

var parallel *bool = flag.Bool("parallel", false, "Enable paralleled parsing")
var output string

type par struct {
	idx int
	url string
}

func main() {
	flag.StringVar(&output, "output", "", "Output file (default: /dev/stdout)")
	flag.StringVar(&output, "o", "", "Output file (defualt: /dev/stdout) (shorthand)")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s <Google Photo Album URL>:\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()
	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	out := os.Stdout
	if len(output) > 0 {
		file, err := os.OpenFile(output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		out = file
	}
	mlist, title := listAlbum(flag.Arg(0))
	fmt.Fprintf(os.Stderr, "title: %s\n", title)
	if *parallel {
		rlist := make([]string, len(mlist))
		rch := make(chan par)
		var wg sync.WaitGroup
		bar := progressbar.New(len(mlist))
		for i, item := range mlist {
			fmt.Fprintf(os.Stderr, "%02d %s\n", i, item)
			wg.Add(1)
			go func(i int, item string) {
				rch <- par{i, sniffer(item)}
				_ = bar.Add(1)
				wg.Done()
			}(i, item)
		}
		go func() { wg.Wait(); close(rch) }()
		for p := range rch {
			rlist[p.idx] = p.url
		}
		fmt.Fprintln(os.Stderr)
		for _, item := range rlist {
			fmt.Fprintln(out, item)
		}
	} else {
		for i, item := range mlist {
			fmt.Fprintf(os.Stderr, "%02d %s\n", i, item)
			fmt.Fprintln(out, sniffer(item))
		}
	}
}
