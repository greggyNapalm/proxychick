## Description
Golang-powered library and command line tool to evaluate proxylist.

![Codacy Badge](https://app.codacy.com/project/badge/Grade/e5fc956874694e83a35d0b4ec16161be)
![codebeat](https://goreportcard.com/badge/github.com/greggyNapalm/proxychick)
## Installation
ProxyChick is available using the standard go get command.

Install by running:

```bash
go get github.com/greggyNapalm/proxychick
```

## Usage

```bash
$ go run cmd/main.go -h
  -c int
    	number of simultaneous HTTP requests(maxConcurrency) (default 300)
  -i string
    	path to the proxylist file (default "STDIN")
  -o string
    	path to the results file (default "STDOUT")
  -p string
    	Proxy protocol. If not specified in list, choose one of http/https/socks4/socks4a/socks5/socks5h (default "http")
  -s	Disable the progress meter
  -t string
    	Target URL (default "https://www.cloudflare.com/cdn-cgi/trace")
  -to int
    	Timeout for entire HTTP request in seconds (default 10)
```

## Results table
| Column name          | Type and Dimention | Description                                                                                                          |
|----------------------|:------------------:|:---------------------------------------------------------------------------------------------------------------------|
| proxy                |       string       | proxy URL that was used in test                                                                                      |
| result               |        bool        | true if the target resource replied in time                                                                          |
| targetStatusCode     |        int         | HTTP status code of terget resource reply                                                                            |
| proxyStatusCode      |        int         | HTTP status code of proxy server reply for "CONNECT" request                                                         |
| latency.ttfb         |        int         | Time passed btw the start of the request and the client receiving the first byte from the target (ms = milliseconds) |
| latency.dnsResolve   |        int         | Time passed btw the start of the DNS lookup and its end (ms)                                                         |
| latency.conn         |        int         | Time passed btw the TCP CONNECT start from client to proxy server and the moment its ESTABLISHED (ms)                    |
| latency.tlsHandShake |        int         | How long it takes to establish TLS session btw client and target (ms)                                                    |
| latency.proxyResp    |        int         | Time passed btw the start of the request and proxy server reply reded (ms)                                               |
| ProxyServIPAddr      |       string       | IPv4 or IPv6 addr of proxy service entry point we are connecting to                                                  |
| ProxyNodeIPAddr      |       string       | IPv4 or IPv6 addr of the proxy exit node                                                                             |
| error                |       string       | Error description if any                                                                                             |