package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
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

func list(src string) (result []string, title string) {
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
		fmt.Fprintln(os.Stderr, "retry...")
	}
}

func main() {
	if len(os.Args) < 2 {
		panic("missing argument")
	}
	mlist, title := list(os.Args[1])
	fmt.Fprintf(os.Stderr, "title: %s\n", title)
	for i, item := range mlist {
		fmt.Fprintf(os.Stderr, "%02d %s\n", i, item)
		fmt.Println(sniffer(item))
	}
}
