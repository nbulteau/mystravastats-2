package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestWaitForServerReadyReturnsTrueWhenHealthResponds(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if !waitForServerReady(ctx, server.URL, time.Millisecond) {
		t.Fatal("expected ready server to be detected")
	}
}

func TestWaitForServerReadyReturnsFalseWhenPortIsClosed(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on random local port: %v", err)
	}
	url := "http://" + listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	if waitForServerReady(ctx, url, time.Millisecond) {
		t.Fatal("expected closed port to stay unavailable")
	}
}
