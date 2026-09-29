package web

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"
)

func servePack(t *testing.T, contentType string, packs ...ComponentsPack) string {
	t.Helper()
	rec := httptest.NewRecorder()
	New().PacksHandler(contentType, packs...).ServeHTTP(rec, httptest.NewRequest("GET", "/x", nil))
	body, err := io.ReadAll(rec.Result().Body)
	if err != nil {
		t.Fatal(err)
	}
	if got := rec.Header().Get("Content-Type"); got != contentType {
		t.Fatalf("Content-Type = %q, want %q", got, contentType)
	}
	return string(body)
}

func TestPacksHandlerBody(t *testing.T) {
	if got, want := servePack(t, "text/javascript", "a()", "b()"), "a();\n\nb();\n\n"; got != want {
		t.Fatalf("javascript body = %q, want %q", got, want)
	}
	if got, want := servePack(t, "text/css", "a{}", "b{}"), "a{}\n\nb{}\n\n"; got != want {
		t.Fatalf("css body = %q, want %q", got, want)
	}
}

// Every presets builder mounts the same assets; the handlers must share one
// body rather than each keeping its own copy.
func TestPacksBodySharedAcrossHandlers(t *testing.T) {
	big := ComponentsPack(strings.Repeat("x", 200<<10)) // spans several hash chunks
	// Built from a different backing string with the same bytes, as
	// HandleMaterialDesignIcons does with its strings.ReplaceAll per builder.
	same := ComponentsPack(strings.Repeat("x", 200<<10))

	first := packsBody("text/javascript", []ComponentsPack{big, "tail()"})
	second := packsBody("text/javascript", []ComponentsPack{same, "tail()"})
	if &first[0] != &second[0] {
		t.Fatal("identical packs built two bodies; want one shared body")
	}
	if cap(first) != len(first) {
		t.Fatalf("body cap %d > len %d: the retained body must be sized exactly", cap(first), len(first))
	}

	// The separator is part of the body, so the same packs served as CSS
	// are a different body.
	css := packsBody("text/css", []ComponentsPack{big, "tail()"})
	if &css[0] == &first[0] {
		t.Fatal("javascript and css bodies were shared although their bytes differ")
	}
	if strings.HasSuffix(string(css), ";\n\n") {
		t.Fatal("css body carries the javascript separator")
	}
}
