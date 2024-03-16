package client

import (
	"context"
	"crypto/tls"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"time"
)

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
			resolvedAddr, _ := net.ResolveTCPAddr("tcp", addr)
			res.ProxyServIPAddr = resolvedAddr.IP.String()
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
			// works only for HTTP Connect proxy(that works only over TCP)
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
			res.RespPayload = "N/A"
		} else {
			res.RespPayload = string(b)
		}
	}
	resp.Body.Close()

	res.TargetStatusCode = resp.StatusCode
	res.Status = true
	return
}
