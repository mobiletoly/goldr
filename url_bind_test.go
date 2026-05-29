// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package goldr

import (
	"net/http"
	"testing"
)

type testRequestBinder struct {
	params []string
}

type testRequestBoundRoute struct {
	value string
}

func (b testRequestBinder) GoldrRouteParams() []string {
	return append([]string(nil), b.params...)
}

func (b testRequestBinder) Bind(value string) testRequestBoundRoute {
	return testRequestBoundRoute{value: value}
}

func TestBindFromRequestUsesLastRouteParam(t *testing.T) {
	r := new(http.Request)
	r.SetPathValue("office_id", "office-1")
	r.SetPathValue("team_id", "team-1")

	got, ok := BindFromRequest(r, testRequestBinder{
		params: []string{"office_id", "team_id"},
	})
	if !ok {
		t.Fatal("BindFromRequest() ok = false, want true")
	}
	if got.value != "team-1" {
		t.Fatalf("BindFromRequest() value = %q, want %q", got.value, "team-1")
	}
}

func TestBindFromRequestReportsMissingRequestState(t *testing.T) {
	tests := []struct {
		name   string
		req    *http.Request
		params []string
	}{
		{
			name:   "nil request",
			req:    nil,
			params: []string{"id"},
		},
		{
			name:   "no params",
			req:    new(http.Request),
			params: nil,
		},
		{
			name:   "missing path value",
			req:    new(http.Request),
			params: []string{"id"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := BindFromRequest(test.req, testRequestBinder{params: test.params})
			if ok {
				t.Fatal("BindFromRequest() ok = true, want false")
			}
			if got != (testRequestBoundRoute{}) {
				t.Fatalf("BindFromRequest() = %#v, want zero value", got)
			}
		})
	}
}
