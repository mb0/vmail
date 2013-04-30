package feeds

import (
	"code.google.com/p/go-charset/charset"
	"code.google.com/p/go-html-transform/h5"
	"code.google.com/p/go-html-transform/html/transform"
	"code.google.com/p/go.net/html"
	"encoding/xml"
	"fmt"
	"github.com/ungerik/go-rss"
	"io"
	"net/http"
)

type Feed struct {
	Channel rss.Channel `xml:"channel"`
}

func (f *Feed) Entry(at int) *Entry {
	return &Entry{f.Channel.Item[at]}
}

func Read(r io.Reader) (*Feed, error) {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charset.NewReader
	var f Feed
	if err := dec.Decode(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

func ReadHttp(url string) (*Feed, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return Read(resp.Body)
}

type Entry struct {
	rss.Item
}

func (e *Entry) Html() string {
	parts, err := h5.PartialFromString(e.Description)
	if err != nil {
		fmt.Println(err)
		return ""
	}
	tree := h5.NewTree(&html.Node{
		Type:       html.DocumentNode,
		FirstChild: parts[0],
	})
	t := transform.New(&tree)
	t.Apply(transform.TransformFunc(imgAlt), "img")
	desc := h5.RenderNodesToString([]*html.Node{t.Doc()})
	return fmt.Sprintf(`<h1><a href="%s">%s</a></h1>
%s
<p>Url: <a href="%s">%s</a></p>`, e.Link, e.Title, desc, e.Link, e.Link)
}

func imgAlt(n *html.Node) {
	var alt string
	for _, a := range n.Attr {
		if a.Key == "alt" {
			alt = a.Val
			break
		}
	}
	if alt == "" {
		return
	}
	p := h5.Element("p", nil, h5.Text(alt))
	if n.NextSibling != nil {
		n.Parent.InsertBefore(p, n.NextSibling)
	} else {
		n.Parent.AppendChild(p)
	}
}
