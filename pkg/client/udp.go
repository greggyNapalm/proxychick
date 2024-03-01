package client

import (
	"bufio"
	"fmt"
	"github.com/greggyNapalm/gost"
	"net"
	"net/url"
	"time"
)

func TestUDPEcho(targetURL *url.URL, proxyURL *url.URL, timeOut int, includeRespPayload bool, debug bool) (res *Result, err error) {
	res = &Result{}
	res.ProxyURL = proxyURL.String()
	res.TargetURL = targetURL.String()
	res.Status = false
	client := &gost.Client{
		Connector:   gost.SOCKS5UDPConnector(proxyURL.User),
		Transporter: gost.TCPTransporter(),
	}
	// XXX: Commented code needed to resolve target ip_addr but FQDN works fine in my tests
	//resolveStarted := time.Now()
	//targetIPs, err := net.LookupIP(targetURL.Host)
	//if err != nil {
	//	return
	//}
	//res.Latency.DNSresolve = int(time.Since(resolveStarted).Milliseconds())
	targetPort, err := net.LookupPort("tcp", targetURL.Scheme)
	if err != nil {
		return
	}
	//targetEntryPoint := fmt.Sprintf("%s:%d", targetIPs[0], targetPort)
	targetEntryPoint := fmt.Sprintf("%s:%d", targetURL.Host, targetPort)
	AllStarted := time.Now()
	conn, err := client.Dial(proxyURL.Host, gost.TimeoutDialOption(time.Duration(timeOut)*time.Second))
	if err != nil {
		return
	}
	res.Latency.ProxyResp = int(time.Since(AllStarted).Milliseconds())
	addr, _ := net.ResolveTCPAddr("tcp", conn.RemoteAddr().String())
	res.ProxyServIPAddr = addr.IP.String()
	defer conn.Close()

	udpConn, err := client.Connect(conn, targetEntryPoint, gost.TimeoutConnectOption(time.Duration(timeOut)*time.Second))
	if err != nil {
		return
	}
	udpConn.SetDeadline(time.Now().Add(time.Duration(timeOut) * time.Second))
	defer udpConn.Close()
	udpConn.Write([]byte("Hello from ProxyChick"))
	resp := make([]byte, 1024)
	n, err := bufio.NewReader(udpConn).Read(resp)
	if err != nil {
		return
	}
	// It's TimeToLastByte, but they fits in one datagram, so it good enough for the test.
	res.Latency.TTFB = int(time.Since(AllStarted).Milliseconds())
	if includeRespPayload && n > 0 {
		res.RespPayload = string(resp[:n])
	}
	if debug {
		fmt.Println("Connection to proxy: TCP", udpConn.RemoteAddr(), "Connection to target: UDP", targetEntryPoint, "reply:", res.RespPayload)
	}
	res.Status = true
	return
}
