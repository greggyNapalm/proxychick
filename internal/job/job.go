package job

import (
	"github.com/greggyNapalm/proxychick/internal/httpx"
	"log"
	url "net/url"
	"slices"
	"strconv"
	"strings"
)

type PListEvanJobCfg struct {
	MaxConcurrency int
	TargetURL      url.URL
	TimeOut        int
}

func AdaptRowProxyStr(prxStr string, prxProtocol string) (parsedURL *url.URL, err error) {
	prxChemas := []string{"http", "https", "socks4", "socks4a", "socks5", "socks5h"}
	sSplited := strings.Split(prxStr, ":")
	var prxURLFormated string
	//	err = errors.New("Input proxylist: unknown format. Please use one og the follow: login:password@host:port or host:port:login:password")
	err = proxyURLFormatError

	if slices.Contains(prxChemas, sSplited[0]) { // str starts with protocol
		// complete URL with the protocol scheme
		prxURLFormated = prxStr
	} else if _, err := strconv.Atoi(sSplited[2]); err == nil { // last field is port number
		//login:password@host:port format
		prxURLFormated = prxProtocol + "://" + prxStr
	} else if _, err := strconv.Atoi(sSplited[1]); err == nil { // 2nd field is port number
		//host:port:login:password format
		prxURLFormated = prxProtocol + "://" + sSplited[2] + ":" + sSplited[3] + "@" + sSplited[0] + ":" + sSplited[1]
	}
	if prxURLFormated != "" {
		parsedURL, err = url.Parse(prxURLFormated)
		if err != nil {
			log.Fatal("Can't parse Proxy URL:" + prxURLFormated)
			panic(err)
		}
	}
	return
}

func EvaluateProxyList(prxURLs []*url.URL, cfg *PListEvanJobCfg, ch chan httpx.Result) error {
	chTxConnPool := make(chan struct{}, cfg.MaxConcurrency)
	for i := 0; i < cfg.MaxConcurrency; i++ {
		chTxConnPool <- struct{}{}
	}

	for _, prxURL := range prxURLs {
		<-chTxConnPool
		go func(url url.URL) {
			res, _ := httpx.TestHTTP(&cfg.TargetURL, &url, cfg.TimeOut, false)
			ch <- *res
			chTxConnPool <- struct{}{}
		}(*prxURL)
	}
	return nil
}
