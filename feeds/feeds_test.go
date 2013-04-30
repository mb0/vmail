package feeds

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

/*
active rss feeds are:

http://feeds.feedburner.com/tedtalks_video
http://feeds.wnyc.org/radiolab
http://www.cbc.ca/podcasting/includes/wiretap.xml
http://www.tagesschau.de/xml/rss2
http://xkcd.com/rss.xml
*/
func checkFeed(feed *Feed) error {
	expect := `<h1><a href="http://xkcd.com/1187/">Aspect Ratio</a></h1>
<img src="http://imgs.xkcd.com/comics/aspect_ratio.png" title="I&#39;m always disappointed when &#39;Anamorphic Widescreen&#39; doesn&#39;t refer to a widescreen Animorphs movie." alt="I&#39;m always disappointed when &#39;Anamorphic Widescreen&#39; doesn&#39;t refer to a widescreen Animorphs movie."/><p>I&amp;#39;m always disappointed when &amp;#39;Anamorphic Widescreen&amp;#39; doesn&amp;#39;t refer to a widescreen Animorphs movie.</p>
<p>Url: <a href="http://xkcd.com/1187/">http://xkcd.com/1187/</a></p>`
	if got := feed.Entry(0).Html(); got != expect {
		return fmt.Errorf("expected content got %s", got)
	}
	return nil
}
func TestRead(t *testing.T) {
	f, err := os.Open("testdata/xkcd.rss.xml")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	feed, err := Read(f)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s - %s\n", feed.Channel.Title, feed.Channel.Link)
	err = checkFeed(feed)
	if err != nil {
		t.Error(err)
	}
}

func serveFeed(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open("testdata/xkcd.rss.xml")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Type", "application/xml+rss")
	io.Copy(w, f)
}
func testServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(serveFeed))
}
func TestReadHttp(t *testing.T) {
	s := testServer()
	defer s.Close()
	feed, err := ReadHttp(s.URL)
	if err != nil {
		t.Error(err)
		return
	}
	err = checkFeed(feed)
	if err != nil {
		t.Error(err)
	}
}
