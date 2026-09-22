package main

import (
	"strings"
	"testing"
)

func TestConnectToDB_MissingDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")

	s, err := ConnectToDB()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if s != nil {
		t.Fatal("expected nil storage")
	}

	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Fatalf("expected DATABASE_URL error, got %v", err)
	}
}

func TestConnectToDB_InvalidDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "://invalid")

	s, err := ConnectToDB()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if s != nil {
		t.Fatal("expected nil storage")
	}
}
