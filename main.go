package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ipipdotnet/ipdb-go"
)

func GetIpLeading(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if p := recover(); p != nil {
			log.Printf("Panic: %v", p)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			fmt.Fprintf(w, "")
		}
	}()

	var ip string

	headers := r.Header
	if len(headers["X-Forwarded-For"]) > 0 {
		xffs := strings.Split(headers["X-Forwarded-For"][0], ",")
		if len(xffs) > 0 {
			ip = strings.TrimSpace(xffs[0])
		} else {
			panic("empty x-forwarded-for")
		}
	} else if len(headers["X-Real-Ip"]) > 0 {
		ip = headers["X-Real-Ip"][0]
	} else {
		if r.RemoteAddr == "" {
			panic("no ip detect")
		} else {
			if r.RemoteAddr[0] == '[' {
				// ipv6 address [::1]:8080
				ip = r.RemoteAddr[1:strings.Index(r.RemoteAddr, "]")]
			} else {
				ip = r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
			}
		}
	}

	log.Printf("[ User IP ] %s", ip)
	// ip = "13.112.109.30"

	var res string
	var err error

	for _, f := range []func(string) (string, error){
		GetIpByDB,
		GetIpBy3rd,
	} {
		res, err = f(ip)
		if err == nil {
			break
		}
	}

	if res == "" && err != nil {
		panic(fmt.Sprintf("%s", err))
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintf(w, "%s\n", res)

	// print all headers
	fmt.Println()
	for k, v := range headers {
		fmt.Fprintf(w, "%s: %+v\n", k, v)
	}
}

func GetIPDBPath() string {
	path := "."
	if os.Getenv("GEOIP_WORKPATH") != "" {
		path = os.Getenv("GEOIP_WORKPATH")
	}
	filename := "ipipfree.ipdb"
	return filepath.Join(path, filename)
}

func GetIpByDB(ip string) (rs string, e error) {
	defer func() {
		if p := recover(); p != nil {
			e = errors.New(fmt.Sprintf("%v", p))
		}
	}()

	db, err := ipdb.NewCity(GetIPDBPath())
	if err != nil {
		panic(err)
	}

	addrs, err := db.Find(ip, "CN")
	if err != nil {
		panic(err)
	}
	if len(addrs) <= 0 {
		panic("no address found")
	}

	rs = fmt.Sprintf(
		"当前 IP：%s  来自于：%s",
		ip, strings.Join(addrs, " "))
	log.Printf("[ DB ] %s", rs)
	return rs, nil
}

func GetIpBy3rd(ip string) (rs string, e error) {
	defer func() {
		if p := recover(); p != nil {
			e = errors.New(fmt.Sprintf("%s", p))
		}
	}()

	url := fmt.Sprintf("http://freeapi.ipip.net/%s", ip)
	resp_body := tryRequestWithNums(url)

	var addrs []string
	err := json.Unmarshal(resp_body, &addrs)
	if err != nil {
		panic(fmt.Sprintf("json.Unmarshal(): %s", err))
	}
	if len(addrs) <= 0 {
		panic("no address found")
	}

	rs = fmt.Sprintf(
		"当前 IP：%s  来自于：%s",
		ip, strings.Join(addrs, " "))
	log.Printf("[ ipip ] %s", rs)
	return rs, nil
}

func tryRequestWithNums(url string) []byte {
	timeout := time.Second * time.Duration(3)

	for i := 0; i < 3; i++ {
		httpCode, resp := httpRequestGet(url, timeout)
		if httpCode == 200 {
			return resp
		}
	}

	return []byte{}
}

func httpRequestGet(
	url string, timeout time.Duration) (int, []byte) {

	var req *http.Request
	var resp *http.Response
	var client *http.Client
	var text []byte
	var err error
	var err_msg string

	req, err = http.NewRequest("GET", url, nil)
	if err != nil {
		err_msg = fmt.Sprintf("http.NewRequest(): %s", err)
		panic(err_msg)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client = &http.Client{
		Timeout: timeout,
	}

	resp, err = client.Do(req)
	if err != nil {
		err_msg = fmt.Sprintf("client.Do(): %s", err)
		panic(err_msg)
	}

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return resp.StatusCode, []byte("")
	}

	text, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		err_msg = fmt.Sprintf("ioutil.ReadAll(): %s", err)
		panic(err_msg)
	}

	return resp.StatusCode, text
}

func main() {
	args := os.Args
	if len(args) != 2 {
		fmt.Fprint(os.Stderr, "wrong argument: only 1 arguments.\n")
		os.Exit(-1)
	}

	port, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Fprint(os.Stderr, "wrong argument: Port should be INT.\n")
		os.Exit(-1)
	}

	http.HandleFunc("/getIpInfo", GetIpLeading)

	log.Printf("Listening on Port :%d", port)
	err = http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal(err)
	}
}
