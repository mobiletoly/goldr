package hx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBooleanRequestHelpers(t *testing.T) {
	tests := []struct {
		name   string
		header string
		helper func(*http.Request) bool
	}{
		{name: "request", header: HeaderRequest, helper: IsRequest},
		{name: "boosted", header: HeaderBoosted, helper: IsBoosted},
		{name: "history restore", header: HeaderHistoryRestoreRequest, helper: IsHistoryRestoreRequest},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			for _, value := range []string{"", "false", "TRUE", "True"} {
				request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
				request.Header.Set(test.header, value)
				if test.helper(request) {
					t.Fatalf("%s(%q) = true, want false", test.name, value)
				}
			}

			request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
			request.Header.Set(test.header, "true")
			if !test.helper(request) {
				t.Fatalf("%s(true) = false, want true", test.name)
			}
		})
	}
}

func TestRequestValueHelpers(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/", nil)
	request.Header.Set(HeaderCurrentURL, "https://example.test/users")
	request.Header.Set(HeaderPrompt, "confirmed")
	request.Header.Set(HeaderTarget, "users-table")
	request.Header.Set(HeaderTrigger, "save-button")
	request.Header.Set(HeaderTriggerName, "save")

	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "current url", got: CurrentURL(request), want: "https://example.test/users"},
		{name: "prompt", got: Prompt(request), want: "confirmed"},
		{name: "target", got: Target(request), want: "users-table"},
		{name: "trigger id", got: TriggerID(request), want: "save-button"},
		{name: "trigger name", got: TriggerName(request), want: "save"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestResponseStringSetters(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
		setter func(http.ResponseWriter, string)
	}{
		{name: "location", header: HeaderLocation, value: "/dashboard", setter: Location},
		{name: "push url", header: HeaderPushURL, value: "/users", setter: PushURL},
		{name: "redirect", header: HeaderRedirect, value: "/login", setter: Redirect},
		{name: "replace url", header: HeaderReplaceURL, value: "/settings", setter: ReplaceURL},
		{name: "reselect", header: HeaderReselect, value: "#dialog", setter: Reselect},
		{name: "retarget", header: HeaderRetarget, value: "#form-errors", setter: Retarget},
		{name: "reswap", header: HeaderReswap, value: "outerHTML", setter: Reswap},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.setter(recorder, test.value)

			if got := recorder.Header().Get(test.header); got != test.value {
				t.Fatalf("%s header = %q, want %q", test.name, got, test.value)
			}
		})
	}
}

func TestResponseFixedValueSetters(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
		setter func(http.ResponseWriter)
	}{
		{name: "prevent push url", header: HeaderPushURL, want: "false", setter: PreventPushURL},
		{name: "refresh", header: HeaderRefresh, want: "true", setter: Refresh},
		{name: "prevent replace url", header: HeaderReplaceURL, want: "false", setter: PreventReplaceURL},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.setter(recorder)

			if got := recorder.Header().Get(test.header); got != test.want {
				t.Fatalf("%s header = %q, want %q", test.name, got, test.want)
			}
		})
	}
}

func TestTriggerHeadersSetSingleAndMultipleEvents(t *testing.T) {
	tests := []struct {
		name   string
		header string
		setter func(http.ResponseWriter, ...string)
	}{
		{name: "trigger", header: HeaderTrigger, setter: Trigger},
		{name: "trigger after settle", header: HeaderTriggerAfterSettle, setter: TriggerAfterSettle},
		{name: "trigger after swap", header: HeaderTriggerAfterSwap, setter: TriggerAfterSwap},
	}

	for _, test := range tests {
		t.Run(test.name+"/single", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.setter(recorder, "user:saved")

			if got := recorder.Header().Get(test.header); got != "user:saved" {
				t.Fatalf("%s header = %q, want %q", test.name, got, "user:saved")
			}
		})

		t.Run(test.name+"/multiple", func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.setter(recorder, "a", "b")

			if got := recorder.Header().Get(test.header); got != "a, b" {
				t.Fatalf("%s header = %q, want %q", test.name, got, "a, b")
			}
		})
	}
}

func TestTriggerHeadersWithNoEventsLeaveHeaderAbsent(t *testing.T) {
	tests := []struct {
		name   string
		header string
		setter func(http.ResponseWriter, ...string)
	}{
		{name: "trigger", header: HeaderTrigger, setter: Trigger},
		{name: "trigger after settle", header: HeaderTriggerAfterSettle, setter: TriggerAfterSettle},
		{name: "trigger after swap", header: HeaderTriggerAfterSwap, setter: TriggerAfterSwap},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			test.setter(recorder)

			if got := recorder.Header().Get(test.header); got != "" {
				t.Fatalf("%s header = %q, want empty", test.name, got)
			}
		})
	}
}
