package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHomeHandler(t *testing.T) {
	req, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HomeHandler)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("home handler returned wrong status code: got %v, want %v", status, http.StatusOK)
	}

	expected := "Welcome to Linguatron!"
	if rr.Body.String() != expected {
		t.Errorf("home handler returned unexpected body: Got %v, want %v", rr.Body.String(), expected)
	}
}
