package job

import (
	"github.com/greggyNapalm/proxychick/pkg/httpx"
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
	//	fmt.Println("prxStr:", prxStr)
	//	fmt.Println(sSplited)
	var prxURLFormated string
	err = proxyURLFormatError

	if len(sSplited) < 3 {
		return
	} else if slices.Contains(prxChemas, sSplited[0]) { // str starts with protocol
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
			return
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
			res, err := httpx.TestHTTP(&cfg.TargetURL, &url, cfg.TimeOut, false)
			res.Error = httpx.PChickError{err}
			if ch != nil {
				ch <- *res
			}
			chTxConnPool <- struct{}{}
		}(*prxURL)
	}
	return nil
}
