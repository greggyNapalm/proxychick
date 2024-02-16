package httpx

import (
	"crypto/tls"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"
)

type Latency struct {
	TTFB         int `csv:"ttfb",json:"ttfb""`
	DNSresolve   int `csv:"dnsResolve",json:"dnsResolve"`
	Connect      int `csv:"conn",json:"connect"`
	TLSHandshake int `csv:"tlsHandShake",json:"tlsHandShake"`
}

type PChickError struct {
	Err error
}

func (err *PChickError) MarshalCSV() (string, error) {
	if err.Err == nil {
		return "", nil
	}
	return err.Err.Error(), nil
}

type Result struct {
	ProxyURL       string      `csv:"proxy",json:"proxy"`
	Status         bool        `csv:"result",json:"result"`
	TargetURL      string      `csv:"-",json:"endpoint"`
	RespStatusCode int         `csv:"targetStatus",json:"targetStatus"`
	RespBody       string      `csv:"-",json:"-"`
	Latency        Latency     `csv:"latency",json:"latency"`
	Error          PChickError `csv:"error",json:"error"`
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
	}
	AllStarted = time.Now()
	if resp, err = transport.RoundTrip(req); err != nil {
		res.RespStatusCode = 0 // JQuery and YandexTank(Phantom) do the same for transport layer errors
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

	res.RespStatusCode = resp.StatusCode
	res.Status = true
	return
}
