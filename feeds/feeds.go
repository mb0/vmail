package feeds

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/hobeone/go-html-transform/h5"
	"github.com/hobeone/go-html-transform/html/transform"
	"golang.org/x/net/html"
	"golang.org/x/text/encoding/htmlindex"
	txttransform "golang.org/x/text/transform"
)

func Read(r io.Reader) (*Feed, error) {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = newReaderLabel
	var f Feed
	if err := dec.Decode(&f); err != nil {
		return nil, err
	}
	return &f, nil
}

func newReaderLabel(label string, in io.Reader) (io.Reader, error) {
	enc, _ := htmlindex.Get(label)
	if enc != nil {
		return nil, fmt.Errorf("unsupported charset: %q", label)
	}
	return txttransform.NewReader(in, enc.NewDecoder()), nil
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
	Item
}

func renderHtml(r io.Reader, w io.Writer) error {
	parts, err := h5.Partial(r)
	if err != nil {
		return err
	}
	node := &html.Node{Type: html.DocumentNode}
	for _, p := range parts {
		node.AppendChild(p)
	}
	tree := h5.NewTree(node)
	t := transform.New(&tree)
	t.Apply(transform.TransformFunc(imgAlt), "img")
	return h5.RenderNodes(w, []*html.Node{t.Doc()})
}

func (e *Entry) Html() (io.Reader, error) {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "<h1><a href=\"%s\">%s</a></h1>\n", e.Link, e.Title)
	if e.Content != "" {
		renderHtml(strings.NewReader(e.Content), &buf)
	} else if e.Encoded != "" {
		renderHtml(strings.NewReader(e.Encoded), &buf)
	} else {
		renderHtml(strings.NewReader(e.Description), &buf)
	}
	fmt.Fprintf(&buf, "\n<p>Url: <a href=\"%s\">%s</a></p>", e.Link, e.Link)
	if url := e.Enclosure.URL; url != "" {
		fmt.Fprintf(&buf, "\n<p>Enclosure: <a href=\"%s\">%s</a></p>", url, url)
	}
	return &buf, nil
}

func imgAlt(n *html.Node) {
	var alt string
	for _, a := range n.Attr {
		if a.Key == "alt" {
			alt = html.UnescapeString(a.Val)
			break
		}
	}
	if alt == "" {
		return
	}
	p := h5.Element("p", nil, &html.Node{
		Data: alt,
		Type: html.TextNode,
	})
	if n.NextSibling != nil {
		n.Parent.InsertBefore(p, n.NextSibling)
	} else {
		n.Parent.AppendChild(p)
	}
}
