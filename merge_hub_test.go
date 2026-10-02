package web_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	h "github.com/theplant/htmlgo"

	. "github.com/qor5/web/v3"
	"github.com/qor5/web/v3/multipartestutils"
)

// Two pages merging one hub (a presets model's listing and detail), then a
// registration on the hub after they are mounted: each page must still answer
// its own __reload__. Appending onto the hub's slice put the pages' reload in
// the hub's spare capacity, and the late registration overwrote it.
func TestMergeHubKeepsReloadAfterLaterHubRegistration(t *testing.T) {
	noop := func(ctx *EventContext) (r EventResponse, err error) { return }

	var hub EventsHub
	// Three funcs leave the hub's slice at len 3, cap 4: one spare slot.
	hub.RegisterEventFunc("a", noop)
	hub.RegisterEventFunc("b", noop)
	hub.RegisterEventFunc("c", noop)

	page := func(title string) *PageBuilder {
		return New().Page(func(ctx *EventContext) (r PageResponse, err error) {
			r.Body = h.H1(title)
			return
		}).MergeHub(&hub)
	}
	listing := page("Listing")
	detail := page("Detail")

	hub.RegisterEventFunc("late", noop)

	for name, p := range map[string]*PageBuilder{"Listing": listing, "Detail": detail} {
		w := httptest.NewRecorder()
		r := multipartestutils.NewMultipartBuilder().
			EventFunc(ReloadEventFuncID).BuildEventFuncRequest()
		p.ServeHTTP(w, r)

		if w.Code != 200 {
			t.Fatalf("%s: %s answered %d: %s", name, ReloadEventFuncID, w.Code, w.Body.String())
		}
		if !strings.Contains(w.Body.String(), name) {
			t.Errorf("%s: reload did not render the page: %s", name, w.Body.String())
		}
	}
}
