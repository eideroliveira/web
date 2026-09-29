package web

import (
	"bytes"
	"crypto/sha256"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/NYTimes/gziphandler"
	h "github.com/theplant/htmlgo"
)

type Builder struct {
	EventsHub
	layoutFunc LayoutFunc
}

func New() (b *Builder) {
	b = new(Builder)
	b.layoutFunc = defaultLayoutFunc
	return
}

func (b *Builder) LayoutFunc(mf LayoutFunc) (r *Builder) {
	if mf == nil {
		panic("layout func is nil")
	}
	b.layoutFunc = mf
	return b
}

func (p *Builder) EventFuncs(vs ...interface{}) (r *Builder) {
	p.addMultipleEventFuncs(vs...)
	return p
}

type ComponentsPack string

var startTime = time.Now()

func PacksHandler(contentType string, packs ...ComponentsPack) http.Handler {
	return Default.PacksHandler(contentType, packs...)
}

func (b *Builder) PacksHandler(contentType string, packs ...ComponentsPack) http.Handler {
	bodyBytes := packsBody(contentType, packs)

	return gziphandler.GzipHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, r, "", startTime, bytes.NewReader(bodyBytes))
	}))
}

// packBodies holds one copy of every distinct body PacksHandler has built,
// keyed by its SHA-256.
//
// Every presets builder mounts the same main.js, vue.js and Material Design
// Icons assets under its own prefix, and an application with ten builders used
// to keep ten copies of each: about 300 MB of heap, two thirds of it, for
// bytes that never change. A body is only ever read (bytes.NewReader in the
// handler above), so the handlers can share it.
var packBodies sync.Map // [sha256.Size]byte -> []byte

// packsBody returns the concatenated body of packs, shared with any earlier
// call that produced the same bytes. The digest is taken over the stream the
// body is written from, so a body that is already stored is never built.
func packsBody(contentType string, packs []ComponentsPack) []byte {
	sep := "\n\n"
	if strings.Contains(strings.ToLower(contentType), "javascript") {
		sep = ";\n\n"
	}

	size := 0
	digest := sha256.New()
	// The hash takes no strings, and []byte(pk) would copy a 20 MB pack just
	// to hash it, so the packs go through a small buffer instead.
	chunk := make([]byte, 64<<10)
	write := func(s string) {
		for len(s) > 0 {
			n := copy(chunk, s)
			digest.Write(chunk[:n])
			s = s[n:]
		}
	}
	for _, pk := range packs {
		size += len(pk) + len(sep)
		write(string(pk))
		write(sep)
	}
	var key [sha256.Size]byte
	digest.Sum(key[:0])
	if body, ok := packBodies.Load(key); ok {
		return body.([]byte)
	}

	// Sized exactly: a bytes.Buffer grown by doubling kept up to twice the
	// body alive for the life of the process.
	body := make([]byte, 0, size)
	for _, pk := range packs {
		body = append(body, pk...)
		body = append(body, sep...)
	}
	stored, _ := packBodies.LoadOrStore(key, body)
	return stored.([]byte)
}

func NoopLayoutFunc(in PageFunc) PageFunc {
	return in
}

func defaultLayoutFunc(in PageFunc) PageFunc {
	return func(ctx *EventContext) (r PageResponse, err error) {
		r, err = in(ctx)
		if r.PageTitle != "" {
			ctx.Injector.Title(r.PageTitle)
		}
		if err != nil {
			panic(err)
		}
		r.Body = h.HTMLComponents{
			h.RawHTML("<!DOCTYPE html>\n"),
			h.Tag("html").Children(
				h.Head(
					ctx.Injector.GetHeadHTMLComponent(),
				),
				h.Body(
					h.Div(
						r.Body,
					).Id("app").Attr("v-cloak", true),
					ctx.Injector.GetTailHTMLComponent(),
				).Class("front"),
			).Attr(ctx.Injector.HTMLLangAttrs()...),
		}
		return
	}
}
