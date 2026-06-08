package main

import "testing"

func TestListenAddressUsesDefaultPort(t *testing.T) {
	if got := listenAddress(""); got != ":8080" {
		t.Fatalf("expected default address :8080, got %q", got)
	}
}

func TestListenAddressUsesConfiguredPort(t *testing.T) {
	if got := listenAddress("9090"); got != ":9090" {
		t.Fatalf("expected configured address :9090, got %q", got)
	}
}
