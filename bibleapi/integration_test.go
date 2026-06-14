//go:build integration

package bibleapi

import (
	"context"
	"testing"
)

func TestGetVerseIntegration(t *testing.T) {
	c := NewClient()
	c.Rate = 0

	v, err := c.GetVerse(context.Background(), "john 3:16", "")
	if err != nil {
		t.Fatalf("GetVerse: %v", err)
	}
	if v.Reference == "" {
		t.Error("Reference is empty")
	}
	if v.Text == "" {
		t.Error("Text is empty")
	}
	if v.TranslationID == "" {
		t.Error("TranslationID is empty")
	}
	t.Logf("Reference: %s", v.Reference)
	t.Logf("Text: %s", v.Text)
	t.Logf("Translation: %s (%s)", v.TranslationName, v.TranslationID)
}

func TestGetVerseKJVIntegration(t *testing.T) {
	c := NewClient()
	c.Rate = 0

	v, err := c.GetVerse(context.Background(), "john 3:16", "kjv")
	if err != nil {
		t.Fatalf("GetVerse KJV: %v", err)
	}
	if v.TranslationID != "kjv" {
		t.Errorf("TranslationID = %q, want kjv", v.TranslationID)
	}
	t.Logf("KJV John 3:16: %s", v.Text)
}

func TestGetPassageIntegration(t *testing.T) {
	c := NewClient()
	c.Rate = 0

	verses, err := c.GetPassage(context.Background(), "romans 8:28-30", "")
	if err != nil {
		t.Fatalf("GetPassage: %v", err)
	}
	if len(verses) == 0 {
		t.Fatal("no verses returned")
	}
	for _, vd := range verses {
		t.Logf("Romans %d:%d — %s", vd.Chapter, vd.Verse, vd.Text)
	}
}
