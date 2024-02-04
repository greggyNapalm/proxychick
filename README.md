# Description
Golang-powered library and command line tool to evaluate proxylist.

# Installation
ProxyChick is available using the standard go get command.

Install by running:
```bash
go get github.com/greggyNapalm/proxychick
```

# Usage
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

# Demo
![1st ver 100 proxies](https://github.com/greggyNapalm/proxychick/blob/main/docs/examples/100proxies-cmd-demo-1st-ver.gif?raw=true)
