package job

import (
	"encoding/json"
	"github.com/greggyNapalm/proxychick/pkg/client"
	"log"
	url "net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

type PListEvanJobCfg struct {
	MaxConcurrency int           `json:"MaxConcurrency"`
	TargetURL      url.URL       `json:"TargetURL"`
	TimeOut        time.Duration `json:"TimeOut"`
	Transport      string        `json:"Transport"`
	Debug          bool          `json:"-"`
}

func (self PListEvanJobCfg) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MaxConcurrency int    `json:"MaxConcurrency"`
		TargetURL      string `json:"TargetURL"`
		TimeOut        int    `json:"TimeOut"`
		Transport      string `json:"Transport"`
	}{
		self.MaxConcurrency,
		self.TargetURL.String(),
		int(self.TimeOut / time.Millisecond),
		self.Transport,
	})
}

type JobMetrics struct {
	Duration             time.Duration `json:"Duration"`
	UniqueExitNodesIPCnt int           `json:"UniqueExitNodesIPCnt"`
	ReqsCnt              int           `json:"ReqsCnt"`
	RespCnt              int           `json:"RespCnt"`
}

func (self JobMetrics) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Duration             int `json:"Duration"`
		UniqueExitNodesIPCnt int `json:"UniqueExitNodesIPCnt"`
	}{
		int(self.Duration / time.Millisecond),
		self.UniqueExitNodesIPCnt,
	})
}

func AdaptRawProxyStr(prxStr string, prxProtocol string) (parsedURL *url.URL, err error) {
	prxChemas := []string{"http", "https", "socks4", "socks4a", "socks5", "socks5h"}
	sSplited := strings.Split(prxStr, ":")
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

func EvaluateProxyList(prxURLs []*url.URL, cfg *PListEvanJobCfg, ch chan client.Result) error {
	chTxConnPool := make(chan struct{}, cfg.MaxConcurrency)
	for i := 0; i < cfg.MaxConcurrency; i++ {
		chTxConnPool <- struct{}{}
	}

	for _, prxURL := range prxURLs {
		<-chTxConnPool
		go func(url url.URL) {
			var res *client.Result
			var err error
			if cfg.Transport == "tcp" {
				res, err = client.TestHTTP(&cfg.TargetURL, &url, cfg.TimeOut, true)
				res.EnrichHTTP(err)
			} else if cfg.Transport == "udp" {
				res, err = client.TestUDPEcho(&cfg.TargetURL, &url, cfg.TimeOut, true, cfg.Debug)
				res.EnrichUdpEcho(err)
			} else {
				return
			}
			if ch != nil {
				ch <- *res
			}
			chTxConnPool <- struct{}{}
		}(*prxURL)
	}
	return nil
}
