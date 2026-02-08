package util

import (
	"crypto/tls"
	"net/http"
	"net/http/cookiejar"
	"time"
)

func InsecureClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{
		Timeout: 9 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Jar: jar,
	}
}
