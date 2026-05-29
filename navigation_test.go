// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"context"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
)

func TestBackHref(t *testing.T) {
	t.Run("returns nearest previous linked step", func(t *testing.T) {
		href, ok := BackHref(NavTrail{
			NavStep("Home", "/"),
			NavStep("Reports", "/reports"),
			CurrentNavStep("Report"),
		})
		if !ok {
			t.Fatal("BackHref ok = false, want true")
		}
		if href != "/reports" {
			t.Fatalf("BackHref href = %q, want /reports", href)
		}
	})

	t.Run("skips current and blank steps", func(t *testing.T) {
		href, ok := BackHref(NavTrail{
			NavStep("Home", "/"),
			{Label: "Section", Href: "   "},
			CurrentNavStep("Current"),
		})
		if !ok {
			t.Fatal("BackHref ok = false, want true")
		}
		if href != "/" {
			t.Fatalf("BackHref href = %q, want /", href)
		}
	})

	t.Run("reports missing href", func(t *testing.T) {
		href, ok := BackHref(NavTrail{
			CurrentNavStep("Current"),
		})
		if ok {
			t.Fatal("BackHref ok = true, want false")
		}
		if href != "" {
			t.Fatalf("BackHref href = %q, want empty", href)
		}
	})
}

func TestQueryValues(t *testing.T) {
	t.Run("copies selected app query values", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), "GET", "/report?view=summary&page=2&tag=a&tag=b&empty=&_goldr_trail=ignored", nil)

		got := QueryValues(r, "view", "tag", "_goldr_trail", "missing", "empty", "view")
		want := url.Values{
			"view":  {"summary"},
			"tag":   {"a", "b"},
			"empty": {""},
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("QueryValues() = %#v, want %#v", got, want)
		}
	})

	t.Run("returns fresh values", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), "GET", "/report?tag=a&tag=b", nil)

		got := QueryValues(r, "tag")
		got.Set("tag", "changed")

		if requestValue := r.URL.Query()["tag"]; !reflect.DeepEqual(requestValue, []string{"a", "b"}) {
			t.Fatalf("request query tag = %#v, want original values", requestValue)
		}
	})

	t.Run("handles nil request", func(t *testing.T) {
		if got := QueryValues(nil, "view"); len(got) != 0 {
			t.Fatalf("QueryValues(nil) = %#v, want empty values", got)
		}
	})
}

func TestNavTrailSelected(t *testing.T) {
	r := httptest.NewRequestWithContext(context.Background(), "GET", "/report?_goldr_trail=ignored", nil)
	selected := WithNavTrailKey(r, "provider-profile")

	if !NavTrailSelected(selected, "provider-profile") {
		t.Fatal("NavTrailSelected() = false, want true")
	}
	if NavTrailSelected(selected, "provider-search") {
		t.Fatal("NavTrailSelected() = true for different key, want false")
	}
	if NavTrailSelected(selected, "") {
		t.Fatal("NavTrailSelected() = true for empty key, want false")
	}
	if NavTrailSelected(r, "provider-profile") {
		t.Fatal("NavTrailSelected() = true for request without selected key, want false")
	}
	if NavTrailSelected(nil, "provider-profile") {
		t.Fatal("NavTrailSelected(nil) = true, want false")
	}
}
