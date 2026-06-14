package bibleapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("request carried no User-Agent")
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0 // no pacing in the test

	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "ok" {
		t.Errorf("body = %q, want %q", body, "ok")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("recovered"))
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0
	c.Retries = 5

	start := time.Now()
	body, err := c.Get(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "recovered" {
		t.Errorf("body = %q after retries", body)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
	if time.Since(start) < 500*time.Millisecond {
		t.Error("retries did not back off")
	}
}

func TestGetVerse(t *testing.T) {
	payload := apiResponse{
		Reference:       "John 3:16",
		Text:            "For God so loved the world...\n",
		TranslationID:   "web",
		TranslationName: "World English Bible",
		Verses: []VerseDetail{
			{BookID: "JHN", BookName: "John", Chapter: 3, Verse: 16, Text: "For God so loved the world...\n"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0
	body, err := c.Get(context.Background(), srv.URL+"/john+3:16")
	if err != nil {
		t.Fatal(err)
	}
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Reference != "John 3:16" {
		t.Errorf("Reference = %q, want John 3:16", resp.Reference)
	}
	if resp.TranslationID != "web" {
		t.Errorf("TranslationID = %q, want web", resp.TranslationID)
	}
}

func TestGetPassageParsing(t *testing.T) {
	payload := apiResponse{
		Reference:       "Romans 8:28-30",
		Text:            "We know that all things work together...",
		TranslationID:   "web",
		TranslationName: "World English Bible",
		Verses: []VerseDetail{
			{BookID: "ROM", BookName: "Romans", Chapter: 8, Verse: 28, Text: "We know that all things work together...\n"},
			{BookID: "ROM", BookName: "Romans", Chapter: 8, Verse: 29, Text: "For whom he foreknew...\n"},
			{BookID: "ROM", BookName: "Romans", Chapter: 8, Verse: 30, Text: "Whom he predestined...\n"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer srv.Close()

	c := NewClient()
	c.Rate = 0
	body, err := c.Get(context.Background(), srv.URL+"/romans+8:28-30")
	if err != nil {
		t.Fatal(err)
	}
	var resp apiResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Verses) != 3 {
		t.Errorf("len(Verses) = %d, want 3", len(resp.Verses))
	}
	if resp.Verses[0].Verse != 28 {
		t.Errorf("first verse number = %d, want 28", resp.Verses[0].Verse)
	}
}
