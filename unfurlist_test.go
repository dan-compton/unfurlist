package unfurlist

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestOpenGraph(t *testing.T) {
	result := doRequest("/?content=Test+http://techcrunch.com/2015/11/09/basic-income-createathon/", t)

	want := "Robots To Eat All The Jobs? Hackers, Policy Wonks Collaborate On A Basic Income Createathon This\u00a0Weekend"
	if result[0].Title != want {
		t.Errorf("Title not valid, %q != %q", want, result[0].Title)
	}

	want = "https://tctechcrunch2011.files.wordpress.com/2015/11/basic-income-createathon.jpg?w=764\u0026h=400\u0026crop=1"
	if result[0].Image != want {
		t.Errorf("Image not valid, %q != %q", want, result[0].Title)
	}
}

func TestOpenGraphTwitter(t *testing.T) {
	result := doRequest("/?content=Test+https://twitter.com/amix3k/status/679355208091181056", t)

	want := "Help a family out of hunger and poverty"
	if !strings.Contains(result[0].Title, want) {
		t.Errorf("Title not valid, %q != %q", want, result[0].Title)
	}
}

func TestOembed(t *testing.T) {
	result := doRequest("/?content=Test+https://www.youtube.com/watch?v=Ey8FzGECjFA", t)

	want := "Jony Ive, J.J. Abrams, and Brian Grazer on Inventing Worlds in a Changing One - FULL CONVERSATION"
	if result[0].Title != want {
		t.Errorf("Title not valid, %q != %q", want, result[0].Title)
	}

	want = "https://i.ytimg.com/vi/Ey8FzGECjFA/hqdefault.jpg"
	if result[0].Image != want {
		t.Errorf("Image not valid, %q != %q", want, result[0].Title)
	}

	want = "video"
	if result[0].Type != want {
		t.Errorf("Type not valid, %q != %q", want, result[0].Title)
	}
}

func TestHtml(t *testing.T) {
	result := doRequest("/?content=https://news.ycombinator.com/", t)

	want := "Hacker News"
	if result[0].Title != want {
		t.Errorf("Title not valid, %q != %q", want, result[0].Title)
	}

	want = ""
	if result[0].Image != want {
		t.Errorf("Image not valid, %q != %q", want, result[0].Title)
	}

	want = "website"
	if result[0].Type != want {
		t.Errorf("Type not valid, %q != %q", want, result[0].Type)
	}
}

func doRequest(url string, t *testing.T) []unfurlResult {
	pp := newPipePool()
	defer pp.Close()
	go http.Serve(pp, http.HandlerFunc(replayHandler))
	config := UnfurlConfig{
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				Dial:    pp.Dial,
				DialTLS: pp.Dial,
			},
		},
	}
	handler := New(&config)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", url, nil)

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Techcrunch Open graph test didn't return %v", http.StatusOK)
	}

	var result []unfurlResult
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		t.Errorf("Result isn't JSON %v", w.Body.String())
	}

	return result
}

// replayHandler is a http.Handler responding with pre-recorded data
func replayHandler(w http.ResponseWriter, r *http.Request) {
	d, ok := remoteData[r.Host+r.URL.RequestURI()]
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Write([]byte(d))
}

// pipePool implements net.Listener interface and provides a Dial() func to dial
// to this listener
type pipePool struct {
	m           sync.RWMutex
	closed      bool
	serverConns chan net.Conn
}

func newPipePool() *pipePool { return &pipePool{serverConns: make(chan net.Conn)} }

func (p *pipePool) Accept() (net.Conn, error) {
	c, ok := <-p.serverConns
	if !ok {
		return nil, errors.New("listener is closed")
	}
	return c, nil
}

func (p *pipePool) Close() error {
	p.m.Lock()
	defer p.m.Unlock()
	if !p.closed {
		close(p.serverConns)
		p.closed = true
	}
	return nil
}
func (p *pipePool) Addr() net.Addr { return phonyAddr{} }

func (p *pipePool) Dial(network, addr string) (net.Conn, error) {
	p.m.RLock()
	defer p.m.RUnlock()
	if p.closed {
		return nil, errors.New("listener is closed")
	}
	c1, c2 := net.Pipe()
	p.serverConns <- c1
	return c2, nil
}

type phonyAddr struct{}

func (a phonyAddr) Network() string { return "pipe" }
func (a phonyAddr) String() string  { return "pipe" }

//go:generate go run remote-data-update.go
