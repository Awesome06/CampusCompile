package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

func main() {
	host := "login.microsoftonline.com"
	req, err := http.NewRequestWithContext(context.Background(), "GET", "https://8.8.8.8/resolve?name="+host+"&type=A", nil)
	if err != nil {
		fmt.Println("req error", err)
		return
	}
	req.Host = "dns.google"
	dohClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 5 * time.Second,
	}
	resp, err := dohClient.Do(req)
	if err != nil {
		fmt.Println("do error", err)
		return
	}
	defer resp.Body.Close()
	var dohRes struct {
		Answer []struct {
			Type int    `json:"type"`
			Data string `json:"data"`
		} `json:"Answer"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dohRes); err != nil {
		fmt.Println("json error", err)
		return
	}
	for _, ans := range dohRes.Answer {
		if ans.Type == 1 { // A record
			fmt.Println("Found IP:", ans.Data)
			return
		}
	}
	fmt.Println("No A record found", dohRes)
}
