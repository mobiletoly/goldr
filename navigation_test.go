// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

func TestNavigationBack(t *testing.T) {
	t.Run("returns nearest previous linked step", func(t *testing.T) {
		nav := RequestNav{}.NavigationWithTrail(NavTrail{
			NavStep("Home", "/"),
			NavStep("Reports", "/reports"),
			CurrentNavStep("Report"),
		})
		if !nav.Back.OK {
			t.Fatal("Navigation.Back.OK = false, want true")
		}
		if nav.Back.Href != "/reports" {
			t.Fatalf("Navigation.Back.Href = %q, want /reports", nav.Back.Href)
		}
		if nav.Back.Label != "Reports" {
			t.Fatalf("Navigation.Back.Label = %q, want Reports", nav.Back.Label)
		}
	})

	t.Run("skips current and blank steps", func(t *testing.T) {
		nav := RequestNav{}.NavigationWithTrail(NavTrail{
			NavStep("Home", "/"),
			{Label: "Section", Href: "   "},
			CurrentNavStep("Current"),
		})
		if !nav.Back.OK {
			t.Fatal("Navigation.Back.OK = false, want true")
		}
		if nav.Back.Href != "/" {
			t.Fatalf("Navigation.Back.Href = %q, want /", nav.Back.Href)
		}
	})

	t.Run("reports missing href", func(t *testing.T) {
		nav := RequestNav{}.NavigationWithTrail(NavTrail{
			CurrentNavStep("Current"),
		})
		if nav.Back.OK {
			t.Fatal("Navigation.Back.OK = true, want false")
		}
		if nav.Back.Href != "" {
			t.Fatalf("Navigation.Back.Href = %q, want empty", nav.Back.Href)
		}
	})
}

func TestNav(t *testing.T) {
	t.Run("handles nil request", func(t *testing.T) {
		nav := Nav(nil)

		if got := nav.Trail(); got != nil {
			t.Fatalf("Nav(nil).Trail() = %#v, want nil", got)
		}
		if got := nav.TrailKey(); got != "" {
			t.Fatalf("Nav(nil).TrailKey() = %q, want empty", got)
		}
	})

	t.Run("returns empty state without generated request nav", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), "GET", "/report", nil)
		nav := Nav(r)

		if got := nav.Trail(); got != nil {
			t.Fatalf("Nav(r).Trail() = %#v, want nil", got)
		}
		if got := nav.TrailKey(); got != "" {
			t.Fatalf("Nav(r).TrailKey() = %q, want empty", got)
		}
	})

	t.Run("renders static and resolved dynamic steps", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), "GET", "/models/scale-1/firmware", nil)
		r = WithRequestNav(r, "from-inventory", []RouteNav{
			{Label: "Device Management"},
			{Label: "Scales"},
			{Key: "model"},
			{Label: "Firmware"},
		}, []string{"/admin/conndev", "/admin/conndev/scale/models", "/admin/conndev/scale/models/scale-1", "/admin/conndev/scale/models/scale-1/firmware"}, 3)

		nav := Nav(r)
		if got := nav.TrailKey(); got != "from-inventory" {
			t.Fatalf("TrailKey() = %q, want from-inventory", got)
		}
		nav.Resolve("model", "Scale One")

		want := NavTrail{
			NavStep("Device Management", "/admin/conndev"),
			NavStep("Scales", "/admin/conndev/scale/models"),
			NavStep("Scale One", "/admin/conndev/scale/models/scale-1"),
			CurrentNavStep("Firmware"),
		}
		if got := nav.Trail(); !reflect.DeepEqual(got, want) {
			t.Fatalf("Trail() = %#v, want %#v", got, want)
		}
		navigation := nav.Navigation()
		if !reflect.DeepEqual(navigation.Trail, want) {
			t.Fatalf("Navigation().Trail = %#v, want %#v", navigation.Trail, want)
		}
		if !navigation.Back.OK || navigation.Back.Href != "/admin/conndev/scale/models/scale-1" {
			t.Fatalf("Navigation().Back = %#v, want model href", navigation.Back)
		}
		if !navigation.Current.OK || navigation.Current.Label != "Firmware" {
			t.Fatalf("Navigation().Current = %#v, want Firmware", navigation.Current)
		}
	})

	t.Run("omits unresolved dynamic steps and ancestors without href", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), "GET", "/models/scale-1/firmware", nil)
		r = WithRequestNav(r, "", []RouteNav{
			{Label: "Device Management"},
			{Key: "model"},
			{Label: "Firmware"},
		}, []string{"", "/models/scale-1", ""}, 2)

		want := NavTrail{CurrentNavStep("Firmware")}
		if got := Nav(r).Trail(); !reflect.DeepEqual(got, want) {
			t.Fatalf("Trail() = %#v, want %#v", got, want)
		}
	})

	t.Run("resolves href and shares state across copies", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), "GET", "/models/scale-1/firmware", nil)
		r = WithRequestNav(r, "", []RouteNav{
			{Key: "model"},
			{Label: "Firmware"},
		}, []string{"/models/scale-1", ""}, 1)

		nav := Nav(r)
		nav.ResolveHref("model", "Scale One", "/inventory?model=scale-1")
		Nav(r).Resolve("model", "Scale 1")

		want := NavTrail{
			NavStep("Scale 1", "/models/scale-1"),
			CurrentNavStep("Firmware"),
		}
		if got := nav.Trail(); !reflect.DeepEqual(got, want) {
			t.Fatalf("Trail() = %#v, want %#v", got, want)
		}
	})

	t.Run("ignores unknown keys and empty labels", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), "GET", "/models/scale-1", nil)
		r = WithRequestNav(r, "", []RouteNav{{Key: "model"}}, []string{""}, 0)

		nav := Nav(r)
		nav.Resolve("other", "Other")
		nav.Resolve("model", "")

		if got := nav.Trail(); got != nil {
			t.Fatalf("Trail() = %#v, want nil", got)
		}
	})

	t.Run("uses valid return-to only with selected navigation key", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/reports/contoso?_goldr_nav_trail_key=from-analytics&_goldr_return_to=%2Fanalytics%3Frisk%3Dhigh%26page%3D2", nil)
		r = WithRequestNav(r, "from-analytics", []RouteNav{
			{Label: "Analytics"},
			{Label: "Report"},
		}, []string{"/analytics", "/reports/contoso"}, 1)

		nav := Nav(r).Navigation()
		if !nav.Back.OK {
			t.Fatal("Navigation().Back.OK = false, want true")
		}
		if got, want := nav.Back.Href, "/analytics?page=2&risk=high"; got != want {
			t.Fatalf("Navigation().Back.Href = %q, want %q", got, want)
		}
	})

	t.Run("ignores return-to without selected navigation key", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/reports/contoso?_goldr_return_to=%2Fanalytics%3Frisk%3Dhigh", nil)
		r = WithRequestNav(r, "", []RouteNav{
			{Label: "Analytics"},
			{Label: "Report"},
		}, []string{"/analytics", "/reports/contoso"}, 1)

		nav := Nav(r).Navigation()
		if got, want := nav.Back.Href, "/analytics"; got != want {
			t.Fatalf("Navigation().Back.Href = %q, want %q", got, want)
		}
	})

	t.Run("ignores external return-to", func(t *testing.T) {
		for _, target := range []string{"https://example.com/analytics", "//example.com/analytics"} {
			r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/reports/contoso?_goldr_nav_trail_key=from-analytics&_goldr_return_to="+target, nil)
			r = WithRequestNav(r, "from-analytics", []RouteNav{
				{Label: "Analytics"},
				{Label: "Report"},
			}, []string{"/analytics", "/reports/contoso"}, 1)

			nav := Nav(r).Navigation()
			if got, want := nav.Back.Href, "/analytics"; got != want {
				t.Fatalf("Navigation().Back.Href for %q = %q, want %q", target, got, want)
			}
		}
	})

	t.Run("navigation href preserves current URL without nested return-to", func(t *testing.T) {
		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/analytics?risk=high&page=2&_goldr_return_to=%2Fold", nil)
		r = WithRequestNav(r, "", []RouteNav{
			{Label: "Analytics"},
		}, []string{"/analytics"}, 0)

		href := NavigationHref("/reports/contoso?view=summary", "from-analytics", Nav(r).Navigation())
		for _, want := range []string{
			"/reports/contoso?",
			"view=summary",
			"_goldr_nav_trail_key=from-analytics",
			"_goldr_return_to=%2Fanalytics%3Fpage%3D2%26risk%3Dhigh",
		} {
			if !strings.Contains(href, want) {
				t.Fatalf("NavigationHref missing %q: %s", want, href)
			}
		}
		if strings.Contains(href, "%252Fold") || strings.Contains(href, "_goldr_return_to=%2Fold") {
			t.Fatalf("NavigationHref preserved nested return-to: %s", href)
		}
	})
}
