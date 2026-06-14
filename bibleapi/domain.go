package bibleapi

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

// domain.go exposes bibleapi as a kit Domain: a driver that a multi-domain
// host (ant) enables with a single blank import,
//
//	import _ "github.com/tamnd/bibleapi-cli/bibleapi"
//
// exactly as a database/sql program enables a driver with `import _
// "github.com/lib/pq"`.
func init() { kit.Register(Domain{}) }

// Domain is the bibleapi driver. It carries no state.
type Domain struct{}

// Info describes the scheme, hostnames, and binary identity.
func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "bibleapi",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "bibleapi",
			Short:  "A command line for the Bible API.",
			Long: `A command line for bible-api.com.

bibleapi reads public Bible data over plain HTTPS, shapes it into
clean records, and prints output that pipes into the rest of your tools. No API
key, nothing to run alongside it.`,
			Site: Host,
			Repo: "https://github.com/tamnd/bibleapi-cli",
		},
	}
}

// Register installs the client factory and every operation onto app.
func (Domain) Register(app *kit.App) {
	app.SetClient(newClient)

	// verse: fetch a single verse (summary).
	kit.Handle(app, kit.OpMeta{
		Name:    "verse",
		Group:   "read",
		Single:  true,
		Summary: "Fetch a Bible verse by reference",
		URIType: "verse",
		Resolver: true,
		Args:    []kit.Arg{{Name: "ref", Help: "verse reference, e.g. \"john 3:16\""}},
	}, getVerse)

	// passage: fetch a passage and emit each verse individually.
	kit.Handle(app, kit.OpMeta{
		Name:    "passage",
		Group:   "read",
		List:    true,
		Summary: "Fetch a Bible passage and stream each verse",
		URIType: "verse",
		Args:    []kit.Arg{{Name: "ref", Help: "passage reference, e.g. \"romans 8:28-30\""}},
	}, getPassage)
}

// newClient builds the client from the host-resolved config.
func newClient(_ context.Context, cfg kit.Config) (any, error) {
	c := NewClient()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.HTTP.Timeout = cfg.Timeout
	}
	return c, nil
}

// --- inputs ---

type verseInput struct {
	Ref         string  `kit:"arg"       help:"verse reference, e.g. \"john 3:16\""`
	Translation string  `kit:"flag"      help:"translation id (web, kjv, bbe, oeb-us, almeida, rccv)"`
	Client      *Client `kit:"inject"`
}

type passageInput struct {
	Ref         string  `kit:"arg"       help:"passage reference, e.g. \"romans 8:28-30\""`
	Translation string  `kit:"flag"      help:"translation id (web, kjv, bbe, oeb-us, almeida, rccv)"`
	Client      *Client `kit:"inject"`
}

// --- handlers ---

func getVerse(ctx context.Context, in verseInput, emit func(*Verse) error) error {
	v, err := in.Client.GetVerse(ctx, in.Ref, in.Translation)
	if err != nil {
		return mapErr(err)
	}
	return emit(v)
}

func getPassage(ctx context.Context, in passageInput, emit func(*VerseDetail) error) error {
	verses, err := in.Client.GetPassage(ctx, in.Ref, in.Translation)
	if err != nil {
		return mapErr(err)
	}
	for i := range verses {
		if err := emit(&verses[i]); err != nil {
			return err
		}
	}
	return nil
}

// --- Resolver ---

// Classify turns a reference string into a (type, id) pair.
func (Domain) Classify(input string) (uriType, id string, err error) {
	if input == "" {
		return "", "", errs.Usage("empty bibleapi reference")
	}
	return "verse", input, nil
}

// Locate returns the live https URL for a (type, id).
func (Domain) Locate(uriType, id string) (string, error) {
	if uriType != "verse" {
		return "", errs.Usage("bibleapi has no resource type %q", uriType)
	}
	return buildURL(id, ""), nil
}

// mapErr converts a library error into the kit error kind that carries the
// right exit code.
func mapErr(err error) error {
	return err
}
