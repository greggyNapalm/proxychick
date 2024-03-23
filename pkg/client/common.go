package client

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Latency struct {
	TTFB         int `csv:"ttfb",json:"ttfb""`
	DNSresolve   int `csv:"dnsResolve",json:"dnsResolve"`
	Connect      int `csv:"conn",json:"conn"`
	TLSHandshake int `csv:"tlsHandShake",json:"tlsHandShake"`
	ProxyResp    int `csv:"proxyResp",json:"proxyResp"`
}

type PChickError struct {
	Err error
}

func (err *PChickError) Error() string {
	return err.Error()
}

func (err *PChickError) MarshalCSV() (string, error) {
	if err.Err == nil {
		return "", nil
	}
	return strings.ReplaceAll(err.Err.Error(), ",", ";"), nil
}

type URL struct {
	url.URL
}

func (u URL) MarshalCSV() (string, error) {
	return u.String(), nil
}
func (u URL) MarshalJSON() (string, error) {
	return u.String(), nil
}

type Result struct {
	ProxyURL         URL         `csv:"proxy",json:"proxy,string"`
	Status           bool        `csv:"result",json:"result"`
	TargetURL        url.URL     `csv:"-",json:"-"`
	TargetStatusCode int         `csv:"targetStatusCode",json:"targetStatusCode"`
	ProxyStatusCode  int         `csv:"proxyStatusCode",json:"proxyStatusCode"`
	RespPayload      string      `csv:"-",json:"-"`
	ProxyRespHeader  http.Header `csv:"-",json:"-"`
	Latency          Latency     `csv:"latency",json:"latency"`
	ProxyServIPAddr  net.IP      `csv:"ProxyServIPAddr",json:"ProxyServIPAddr"`
	ProxyNodeIPAddr  net.IP      `csv:"ProxyNodeIPAddr",json:"ProxyNodeIPAddr"`
	Error            PChickError `csv:"error",json:"error"`
}

func (res *Result) MarshalJSON() ([]byte, error) {
	errStr, _ := res.Error.MarshalCSV()
	return json.Marshal(struct {
		ProxyURL         string  `json:"proxy"`
		Status           bool    `json:"result"`
		TargetStatusCode int     `json:"targetStatusCode"`
		ProxyStatusCode  int     `json:"proxyStatusCode"`
		Latency          Latency `json:"latency"`
		ProxyServIPAddr  net.IP  `json:"ProxyServIPAddr"`
		ProxyNodeIPAddr  net.IP  `json:"ProxyNodeIPAddr"`
		Error            string  `json:"error"`
	}{
		res.ProxyURL.String(),
		res.Status,
		res.TargetStatusCode,
		res.ProxyStatusCode,
		res.Latency,
		res.ProxyServIPAddr,
		res.ProxyNodeIPAddr,
		errStr,
	})
}

// Enrich test Result with metadata and normilise Error text.
func (res *Result) EnrichHTTP(err error) error {
	res.Error = PChickError{err}
	if res.ProxyStatusCode != 200 {
		if val, ok := res.ProxyRespHeader["Reason"]; ok { // SOAX header detected
			res.Error = PChickError{errors.New("Proxy Error:" + strings.Split(val[0], ";")[0])}
		}
		if val, ok := res.ProxyRespHeader["X-Luminati-Error"]; ok { // Luminati header detected
			res.Error = PChickError{errors.New("Proxy Error:" + val[0])}
		}
	}
	if res.RespPayload != "" {
		if res.TargetURL.String() == "https://www.cloudflare.com/cdn-cgi/trace" {
			for _, val := range strings.Split(res.RespPayload, "\n") {
				if strings.HasPrefix(val, "ip=") {
					res.ProxyNodeIPAddr = net.ParseIP(strings.Split(val, "=")[1])
				}
			}
		} else if res.TargetURL.String() == "https://api.datascrape.tech/latest/ip" {
			res.ProxyNodeIPAddr = net.ParseIP(res.RespPayload)
		}
	}
	return nil
}

func (res *Result) EnrichUdpEcho(err error) error {
	res.Error = PChickError{err}
	if res.RespPayload != "" {
		if err != nil {
			panic(err)
		}
		//TODO: It might be better to rewrite using DNS lookup instead hardocded ip_addr
		if res.TargetURL.Host == "api.datascrape.tech:80" || res.TargetURL.Host == "194.76.46.8:80" {
			payload := map[string]string{}
			json.Unmarshal([]byte(res.RespPayload), &payload)
			res.ProxyNodeIPAddr = net.ParseIP(payload["clent_ip_addr"])
		}
	}
	return nil
}
