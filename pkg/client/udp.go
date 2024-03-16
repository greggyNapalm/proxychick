package client

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/greggyNapalm/gost"
	"net"
	"net/url"
	"time"
)

func TestUDPEcho(targetAddr *string, proxyURL *url.URL, timeOut int, includeRespPayload bool, debug bool) (res *Result, err error) {
	res = &Result{}
	err = errors.New("proxycheck: Failed to establish TCP connetion to Proxy server")
	res.ProxyURL = proxyURL.String()
	res.TargetURL = *targetAddr
	res.Status = false
	client := &gost.Client{
		Connector:   gost.SOCKS5UDPConnector(proxyURL.User),
		Transporter: gost.TCPTransporter(),
	}
	AllStarted := time.Now()
	conn, err := client.Dial(proxyURL.Host, gost.TimeoutDialOption(time.Duration(timeOut)*time.Second))
	if err != nil {
		return res, errors.New("c2p transport: Failed to establish TCP connetion to Proxy server")
	}
	res.Latency.ProxyResp = int(time.Since(AllStarted).Milliseconds())
	addr, _ := net.ResolveTCPAddr("tcp", conn.RemoteAddr().String())
	res.ProxyServIPAddr = addr.IP.String()
	defer conn.Close()

	udpConn, err := client.Connect(conn, *targetAddr, gost.TimeoutConnectOption(time.Duration(timeOut)*time.Second))
	if err != nil {
		return res, errors.New("c2t transport: Failed to establish UDP connection")
	}
	udpConn.SetDeadline(time.Now().Add(time.Duration(timeOut) * time.Second))
	defer udpConn.Close()
	udpConn.Write([]byte("Hello from ProxyChick"))
	resp := make([]byte, 1024)
	n, err := bufio.NewReader(udpConn).Read(resp)
	if err != nil {
		return res, errors.New("c2t transport: Failed to read from UDP socket")
	}
	// It's TimeToLastByte, but they fits in one datagram, so it good enough for the test.
	res.Latency.TTFB = int(time.Since(AllStarted).Milliseconds())
	if includeRespPayload && n > 0 {
		res.RespPayload = string(resp[:n])
	}
	if debug {
		fmt.Println("Connection to proxy: TCP", udpConn.RemoteAddr(), "Connection to target: UDP", *targetAddr, "reply:", res.RespPayload)
	}
	res.Status = true
	return
}
