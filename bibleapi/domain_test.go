package bibleapi

import "testing"

func TestDomainInfo(t *testing.T) {
	info := Domain{}.Info()
	if info.Scheme != "bibleapi" {
		t.Errorf("Scheme = %q, want bibleapi", info.Scheme)
	}
	if len(info.Hosts) == 0 || info.Hosts[0] != Host {
		t.Errorf("Hosts = %v, want [%s]", info.Hosts, Host)
	}
	if info.Identity.Binary != "bibleapi" {
		t.Errorf("Identity.Binary = %q, want bibleapi", info.Identity.Binary)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		in  string
		typ string
		id  string
	}{
		{"john 3:16", "verse", "john 3:16"},
		{"romans 8:28-30", "verse", "romans 8:28-30"},
	}
	for _, tc := range cases {
		typ, id, err := Domain{}.Classify(tc.in)
		if err != nil || typ != tc.typ || id != tc.id {
			t.Errorf("Classify(%q) = (%q, %q, %v), want (%q, %q, nil)",
				tc.in, typ, id, err, tc.typ, tc.id)
		}
	}
}

func TestLocate(t *testing.T) {
	got, err := Domain{}.Locate("verse", "john 3:16")
	want := "https://" + Host + "/john+3:16"
	if err != nil || got != want {
		t.Errorf("Locate = (%q, %v), want (%q, nil)", got, err, want)
	}
}

func TestBuildURL(t *testing.T) {
	cases := []struct {
		ref         string
		translation string
		want        string
	}{
		{"john 3:16", "", "https://bible-api.com/john+3:16"},
		{"john 3:16", "kjv", "https://bible-api.com/john+3:16?translation=kjv"},
		{"romans 8:28-30", "web", "https://bible-api.com/romans+8:28-30?translation=web"},
	}
	for _, tc := range cases {
		got := buildURL(tc.ref, tc.translation)
		if got != tc.want {
			t.Errorf("buildURL(%q, %q) = %q, want %q", tc.ref, tc.translation, got, tc.want)
		}
	}
}
