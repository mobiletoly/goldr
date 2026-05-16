package main

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestJoinSetsNameCookieAndRedirects(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/join", strings.NewReader(url.Values{
		"name": {"Ada"},
	}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	recorder := httptest.NewRecorder()

	exampleHandler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeBody(t, response.Body)
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusSeeOther)
	}
	if got := response.Header.Get("Location"); got != "/chat" {
		t.Fatalf("Location = %q, want /chat", got)
	}
	cookies := response.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "goldr_chat_name" || cookies[0].Value != "Ada" {
		t.Fatalf("cookies = %#v, want goldr_chat_name=Ada", cookies)
	}
}

func TestChatPageRendersMessagesAndSSEConnection(t *testing.T) {
	body := "page render " + time.Now().Format("150405.000000000")
	postMessage(t, "Grace", body)

	request := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/chat", nil)
	request.AddCookie(&http.Cookie{Name: "goldr_chat_name", Value: "Grace"})
	recorder := httptest.NewRecorder()

	exampleHandler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeBody(t, response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
	page, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	for _, want := range []string{`hx-sse:connect="/chat/events?after=`, `hx-indicator="#send-progress"`, "Sign out", "Sending...", "Grace", body} {
		if !strings.Contains(string(page), want) {
			t.Fatalf("page = %q, want %q", page, want)
		}
	}
}

func TestSignOutClearsNameCookieAndRedirects(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/chat/sign-out", nil)
	request.AddCookie(&http.Cookie{Name: "goldr_chat_name", Value: "Ada"})
	recorder := httptest.NewRecorder()

	exampleHandler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeBody(t, response.Body)
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusSeeOther)
	}
	if got := response.Header.Get("Location"); got != "/" {
		t.Fatalf("Location = %q, want /", got)
	}
	cookies := response.Cookies()
	if len(cookies) != 1 || cookies[0].Name != "goldr_chat_name" || cookies[0].MaxAge != -1 {
		t.Fatalf("cookies = %#v, want expired goldr_chat_name", cookies)
	}
}

func TestPostMessageValidatesBody(t *testing.T) {
	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/chat/message", strings.NewReader(url.Values{
		"body": {"   "},
	}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: "goldr_chat_name", Value: "Hedy"})
	recorder := httptest.NewRecorder()

	exampleHandler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeBody(t, response.Body)
	if response.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusUnprocessableEntity)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if !strings.Contains(string(body), "Enter a message.") {
		t.Fatalf("body = %q, want validation error", body)
	}
}

func TestEventStreamReceivesPostedMessage(t *testing.T) {
	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}
	server := &http.Server{
		Handler:           exampleHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		_ = server.Close()
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	streamRequest, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+listener.Addr().String()+"/chat/events", nil)
	if err != nil {
		t.Fatalf("NewRequestWithContext(stream) error = %v", err)
	}

	client := http.Client{}
	streamResponse, err := client.Do(streamRequest)
	if err != nil {
		t.Fatalf("Do(stream) error = %v", err)
	}
	defer closeBody(t, streamResponse.Body)
	if streamResponse.StatusCode != http.StatusOK {
		t.Fatalf("stream status = %d, want %d", streamResponse.StatusCode, http.StatusOK)
	}
	if got := streamResponse.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	message := "streamed " + time.Now().Format("150405.000000000")
	postURL := "http://" + listener.Addr().String() + "/chat/message"
	postRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, postURL, strings.NewReader(url.Values{
		"body": {message},
	}.Encode()))
	if err != nil {
		t.Fatalf("NewRequestWithContext(post) error = %v", err)
	}
	postRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRequest.AddCookie(&http.Cookie{Name: "goldr_chat_name", Value: "Lin"})
	postResponse, err := client.Do(postRequest)
	if err != nil {
		t.Fatalf("Do(post) error = %v", err)
	}
	defer closeBody(t, postResponse.Body)
	if postResponse.StatusCode != http.StatusOK {
		t.Fatalf("post status = %d, want %d", postResponse.StatusCode, http.StatusOK)
	}

	reader := bufio.NewReader(streamResponse.Body)
	var received strings.Builder
	for !strings.Contains(received.String(), message) {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("ReadString() error = %v; received %q", err, received.String())
		}
		received.WriteString(line)
	}
	if !strings.Contains(received.String(), "data: ") {
		t.Fatalf("received = %q, want SSE data lines", received.String())
	}
}

func postMessage(t *testing.T, author, body string) {
	t.Helper()

	request := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/chat/message", strings.NewReader(url.Values{
		"body": {body},
	}.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(&http.Cookie{Name: "goldr_chat_name", Value: author})
	recorder := httptest.NewRecorder()

	exampleHandler().ServeHTTP(recorder, request)

	response := recorder.Result()
	defer closeBody(t, response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusOK)
	}
}

func closeBody(t *testing.T, body io.Closer) {
	t.Helper()
	if err := body.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
