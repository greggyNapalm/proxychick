package httpx

import (
	"context"
	"crypto/tls"
	"errors"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"strings"
	"time"
)

type Latency struct {
	TTFB         int `csv:"ttfb",json:"ttfb""`
	DNSresolve   int `csv:"dnsResolve",json:"dnsResolve"`
	Connect      int `csv:"conn",json:"connect"`
	TLSHandshake int `csv:"tlsHandShake",json:"tlsHandShake"`
	ProxyResp    int `csv:"proxyResp",json:"proxyResp"`
}

type PChickError struct {
	Err error
}

func (err *PChickError) MarshalCSV() (string, error) {
	if err.Err == nil {
		return "", nil
	}
	return strings.ReplaceAll(err.Err.Error(), ",", ";"), nil
}

type Result struct {
	ProxyURL         string      `csv:"proxy",json:"proxy"`
	Status           bool        `csv:"result",json:"result"`
	TargetURL        string      `csv:"-",json:"endpoint"`
	TargetStatusCode int         `csv:"targetStatusCode",json:"targetStatusCode"`
	ProxyStatusCode  int         `csv:"proxyStatusCode",json:"proxyStatusCode"`
	RespBody         string      `csv:"-",json:"-"`
	ProxyRespHeader  http.Header `csv:"-",json:"-"`
	Latency          Latency     `csv:"latency",json:"latency"`
	ProxyServIPAddr  string      `csv:"ProxyServIPAddr",json:"ProxyServIPAddr"`
	ProxyNodeIPAddr  string      `csv:"ProxyNodeIPAddr",json:"ProxyNodeIPAddr"`
	Error            PChickError `csv:"error",json:"error"`
}

// Enrich test Result with metadata and normilise Error text.
func (res *Result) Enrich(err error) error {
	res.Error = PChickError{err}
	if res.ProxyStatusCode != 200 {
		if val, ok := res.ProxyRespHeader["Reason"]; ok { // SOAX header detected
			res.Error = PChickError{errors.New("Proxy Error:" + strings.Split(val[0], ";")[0])}
		}
		if val, ok := res.ProxyRespHeader["X-Luminati-Error"]; ok { // Luminati header detected
			res.Error = PChickError{errors.New("Proxy Error:" + val[0])}
		}
	}
	if res.RespBody != "" {
		if res.TargetURL == "https://www.cloudflare.com/cdn-cgi/trace" {
			for _, val := range strings.Split(res.RespBody, "\n") {
				if strings.HasPrefix(val, "ip=") {
					res.ProxyNodeIPAddr = strings.Split(val, "=")[1]
				}
			}
		} else if res.TargetURL == "https://api.datascrape.tech/latest/ip" {
			res.ProxyNodeIPAddr = res.RespBody
		}
	}
	return nil
}

func TestHTTP(targetURL *url.URL, proxyURL *url.URL, timeOut int, includeRespBody bool) (res *Result, err error) {
	var resp *http.Response
	var AllStarted, DNSStarted, TcpConnStarted, tlsHandStarted time.Time
	res = &Result{}
	res.ProxyURL = proxyURL.String()
	res.TargetURL = targetURL.String()
	res.Status = false

	req, _ := http.NewRequest("GET", targetURL.String(), nil)
	clientTrace := &httptrace.ClientTrace{
		DNSStart: func(info httptrace.DNSStartInfo) {
			DNSStarted = time.Now()
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			res.Latency.DNSresolve = int(time.Since(DNSStarted).Milliseconds())
		},
		ConnectStart: func(network, addr string) {
			TcpConnStarted = time.Now()
		},
		ConnectDone: func(network, addr string, err error) {
			//fmt.Println("network:", network, "addr:", addr)
			res.ProxyServIPAddr = strings.Split(addr, ":")[0]
			res.Latency.Connect = int(time.Since(TcpConnStarted).Milliseconds())
		},
		TLSHandshakeStart: func() {
			tlsHandStarted = time.Now()
		},
		TLSHandshakeDone: func(cs tls.ConnectionState, err error) {
			res.Latency.TLSHandshake = int(time.Since(tlsHandStarted).Milliseconds())
		},
		GotFirstResponseByte: func() {
			res.Latency.TTFB = int(time.Since(AllStarted).Milliseconds())
		},
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), clientTrace))
	transport := http.Transport{
		Proxy:                 http.ProxyURL(proxyURL),
		TLSHandshakeTimeout:   time.Duration(timeOut) * time.Second,
		ResponseHeaderTimeout: time.Duration(timeOut) * time.Second,
		ExpectContinueTimeout: time.Duration(timeOut) * time.Second,
		DisableKeepAlives:     true,
		MaxIdleConns:          0,
		MaxConnsPerHost:       0,
		OnProxyConnectResponse: func(_ context.Context, _ *url.URL, connectReq *http.Request, connectRes *http.Response) error {
			res.Latency.ProxyResp = int(time.Since(AllStarted).Milliseconds())
			res.ProxyStatusCode = connectRes.StatusCode
			res.ProxyRespHeader = connectRes.Header
			return nil
		},
	}
	AllStarted = time.Now()
	if resp, err = transport.RoundTrip(req); err != nil {
		res.TargetStatusCode = 0 // JQuery and YandexTank(Phantom) do the same for transport layer errors
		return
	}

	if includeRespBody && resp.Body != nil {
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			res.RespBody = "N/A"
		} else {
			res.RespBody = string(b)
		}
	}
	resp.Body.Close()

	res.TargetStatusCode = resp.StatusCode
	res.Status = true
	return
}
