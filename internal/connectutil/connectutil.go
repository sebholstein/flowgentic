package connectutil

import "net/http"

// H2CClient is a shared HTTP client configured for unencrypted HTTP/2 (h2c),
// suitable for Connect RPC clients communicating over plaintext.
var H2CClient = &http.Client{
	Transport: &http.Transport{
		Protocols: h2cProtocols(),
	},
}

// H2CServerProtocols returns an *http.Protocols configured for both HTTP/1
// and unencrypted HTTP/2, suitable for Connect RPC servers.
func H2CServerProtocols() *http.Protocols {
	p := new(http.Protocols)
	p.SetHTTP1(true)
	p.SetUnencryptedHTTP2(true)
	return p
}

func h2cProtocols() *http.Protocols {
	p := new(http.Protocols)
	p.SetUnencryptedHTTP2(true)
	return p
}
