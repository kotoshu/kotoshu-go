package kotoshu

import (
	"context"
	"os"
	"testing"
	"time"
)

func baseURLOrSkip(t *testing.T) string {
	url := os.Getenv("KOTOSHU_TEST_URL")
	if url == "" {
		url = "http://localhost:9292"
	}

	c, err := New(url, WithTimeout(5*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := c.Health(ctx); err != nil {
		t.Skipf("no kotoshu server at %s: %v", url, err)
	}
	return url
}

func TestCheckFlagsMisspelled(t *testing.T) {
	url := baseURLOrSkip(t)
	c, _ := New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r, err := c.Check(ctx, "helo wrold", &CheckOptions{Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"helo": true, "wrold": true}
	got := map[string]bool{}
	for _, e := range r.Errors {
		got[e.Word] = true
	}
	for w := range want {
		if !got[w] {
			t.Errorf("expected %q in errors; got %v", w, got)
		}
	}
}

func TestCheckPassesClean(t *testing.T) {
	url := baseURLOrSkip(t)
	c, _ := New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	r, err := c.Check(ctx, "hello world", &CheckOptions{Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Errors) != 0 {
		t.Errorf("expected no errors, got %d", len(r.Errors))
	}
}

func TestSuggest(t *testing.T) {
	url := baseURLOrSkip(t)
	c, _ := New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sugs, err := c.Suggest(ctx, "helo", &SuggestOptions{Language: "en", Max: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(sugs) > 3 {
		t.Errorf("expected <=3 suggestions, got %d", len(sugs))
	}
	found := false
	for _, s := range sugs {
		if s.Word == "hello" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'hello' in suggestions; got %+v", sugs)
	}
}

func TestDetect(t *testing.T) {
	url := baseURLOrSkip(t)
	c, _ := New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	d, err := c.Detect(ctx, "hello world")
	if err != nil {
		t.Fatal(err)
	}
	if d.Language == "" {
		t.Errorf("expected non-empty language; got %+v", d)
	}
}

func TestCorrect(t *testing.T) {
	url := baseURLOrSkip(t)
	c, _ := New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ok, err := c.Correct(ctx, "hello", &CheckOptions{Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Errorf("expected 'hello' to be correct")
	}

	ok, err = c.Correct(ctx, "helo", &CheckOptions{Language: "en"})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Errorf("expected 'helo' to be flagged")
	}
}

func TestLanguages(t *testing.T) {
	url := baseURLOrSkip(t)
	c, _ := New(url)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	langs, err := c.Languages(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, l := range langs {
		if l == "en" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'en' in languages; got %v", langs)
	}
}
