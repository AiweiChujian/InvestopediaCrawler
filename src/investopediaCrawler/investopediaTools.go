package investopediaCrawler

import (
	"context"
	"errors"
	"golang.org/x/net/html"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

func fetchRequest(link string) (*http.Request, error) {
	req, err  := http.NewRequest("GET", link, nil)
	if err == nil {
		req.Header.Set("User-Agent", "Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1)")
	}
	return req, err
}

func FetchLink(link string) (string, error)  {

	req, err := fetchRequest(link)
	if err != nil {
		return "", err
	}
	client := &http.Client{Transport: &http.Transport{DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
		deadline := time.Now().Add(60 * time.Second)
		c, err1 := net.DialTimeout(network, addr, time.Second*30)
		if err1 != nil {
			return nil, err1
		}
		c.SetDeadline(deadline)
		return c, nil
	}}}
	var resp *http.Response

	var reqErr error
	counter := 0
	for {
		resp, reqErr = client.Do(req)
		counter++
		if(reqErr == nil && resp.StatusCode == 200) || counter > 3 {
			break
		}
	}

	if reqErr != nil || resp.StatusCode != 200{
		return "", errors.New("html request failed: "+ link)
	}

	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return  string(body), nil
}

func isExistAtt(node *html.Node, att string) bool {
	for _, attr := range node.Attr {
		if attr.Key == att {
			return true
		}
	}
	return false
}

func trimNodeText(text string) string  {
	return strings.TrimFunc(text, func(r rune) bool {
		switch r {
		case rune('\n'), rune(' '):
			return  true
		default:
			return false
		}
	})
}

func timestampWithDateText(text string) (int64, error) {
	timeString := trimNodeText(text)
	timeLayout := "Updated Jan 2, 2006"
	tmp, err := time.Parse(timeLayout, timeString)
	if err != nil {
		return 0, err
	}
	return tmp.Unix(), nil
}
