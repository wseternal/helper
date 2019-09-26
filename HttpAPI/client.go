package HttpAPI

import (
	"net/http"
	"net/url"
	"time"
)

var (
	DefaultClient *RequestClient
)

func init() {
	DefaultClient, _ = NewRequestClient(time.Second*10, "")
}

// NewRequestClient will always be successful if proxy is empty
func NewRequestClient(timeout time.Duration, proxy string) (c *RequestClient, err error) {
	c = &RequestClient{
		Client: &http.Client{
			Timeout: timeout,
		},
	}

	if len(proxy) > 0 {
		var u *url.URL
		if u, err = url.Parse(proxy); err != nil {
			return nil, err
		}
		c.tr = &http.Transport{
			Proxy:              http.ProxyURL(u),
			ProxyConnectHeader: make(http.Header),
		}
		c.Client.Transport = c.tr
	}
	return c, nil
}
