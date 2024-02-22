## Description
Golang-powered library and command line tool to evaluate proxylist.

[![Codacy Badge](https://app.codacy.com/project/badge/Grade/e5fc956874694e83a35d0b4ec16161be)](https://app.codacy.com/gh/greggyNapalm/proxychick/dashboard)
[![codebeat](https://goreportcard.com/badge/github.com/greggyNapalm/proxychick)](https://goreportcard.com/report/github.com/greggyNapalm/proxychick)
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
| Column name          | Type and Dimention | Description                                                                                                                                      |
|----------------------|:------------------:|:-------------------------------------------------------------------------------------------------------------------------------------------------|
| proxy                |       string       | Proxy URL that was used in test                                                                                                                  |
| result               |        bool        | True if the target resource replied on time with 200 statsu code                                                                                 |
| targetStatusCode     |        int         | Status code of target resource HTTP reply                                                                                                        |
| proxyStatusCode      |        int         | Status code of proxy server HTTP reply for "CONNECT" request                                                                                     |
| latency.ttfb         |        int         | Time between the initial start and receiving the first byte of the response from the target (ms = milliseconds)                                  |
| latency.dnsResolve   |        int         | How long it takes to perform a DNS lookup (ms)                                                                                                   |
| latency.conn         |        int         | Time passed btw the TCP CONNECT starts from the client to the proxy service and the moment the TCP session turns into the ESTABLISHED state (ms) |
| latency.tlsHandShake |        int         | How long it takes to establish TLS session btw client and target (ms)                                                                            |
| latency.proxyResp    |        int         | Time passed btw the start of the request and proxy server reply reded (ms)                                                                       |
| ProxyServIPAddr      |       string       | IPv4 or IPv6 addr of proxy service entry point we are connecting to                                                                              |
| ProxyNodeIPAddr      |       string       | IPv4 or IPv6 addr of the proxy exit node                                                                                                         |
| error                |       string       | Error description if any                                                                                                                         |