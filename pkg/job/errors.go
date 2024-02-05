package job

import "errors"

var (
	proxyURLFormatError = errors.New("proxycheck: Unknown Proxy URL format. Please use one og the follow: login:password@host:port or host:port:login:password")
	transportLayerError = errors.New("proxycheck: Failed to establish TCP connetion to Proxy server")
)
