package site_test

import (
	"encoding/xml"
	"strings"
	"testing"
	"time"

	"github.com/go-sum/foundry/pkg/web/site"
)

// fp returns a pointer to a float64 value, for use in test literals.
func fp(v float64) *float64 { return &v }

// tp returns a pointer to a time.Time value, for use in test literals.
func tp(t time.Time) *time.Time { return &t }

// parsedURLSet is used to round-trip assert sitemap XML output.
type parsedURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	XMLNS   string       `xml:"xmlns,attr"`
	URLs    []parsedURL  `xml:"url"`
}

type parsedURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

func TestBuildSitemap(t *testing.T) {
	fixedTime := time.Date(2024, 3, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name             string
		entries          []site.Entry
		wantXMLHeader    bool
		wantLocs         []string
		wantNoLocs       []string
		wantContains     []string
		wantNotContains  []string
	}{
		{
			name:          "empty entries produces valid XML with empty urlset",
			entries:       []site.Entry{},
			wantXMLHeader: true,
			wantLocs:      nil,
			wantContains: []string{
				`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`,
				`</urlset>`,
			},
		},
		{
			name: "single entry with all fields present",
			entries: []site.Entry{
				{
					Loc:        "https://example.com/",
					ChangeFreq: "weekly",
					LastMod:    tp(fixedTime),
					Priority:   fp(0.8),
				},
			},
			wantXMLHeader: true,
			wantLocs:      []string{"https://example.com/"},
			wantContains: []string{
				"<changefreq>weekly</changefreq>",
				"<lastmod>2024-03-15</lastmod>",
				"<priority>0.8</priority>",
			},
		},
		{
			name: "nil Priority omits priority element",
			entries: []site.Entry{
				{
					Loc:      "https://example.com/about",
					Priority: nil,
				},
			},
			wantXMLHeader: true,
			wantLocs:      []string{"https://example.com/about"},
			wantNotContains: []string{
				"<priority>",
			},
		},
		{
			name: "explicit zero priority emits priority element with 0.0",
			entries: []site.Entry{
				{
					Loc:      "https://example.com/low",
					Priority: fp(0.0),
				},
			},
			wantXMLHeader: true,
			wantLocs:      []string{"https://example.com/low"},
			wantContains: []string{
				"<priority>0.0</priority>",
			},
		},
		{
			name: "nil LastMod omits lastmod element",
			entries: []site.Entry{
				{
					Loc:     "https://example.com/no-date",
					LastMod: nil,
				},
			},
			wantXMLHeader: true,
			wantLocs:      []string{"https://example.com/no-date"},
			wantNotContains: []string{
				"<lastmod>",
			},
		},
		{
			name: "multiple entries all present",
			entries: []site.Entry{
				{Loc: "https://example.com/"},
				{Loc: "https://example.com/about"},
				{Loc: "https://example.com/contact"},
			},
			wantXMLHeader: true,
			wantLocs: []string{
				"https://example.com/",
				"https://example.com/about",
				"https://example.com/contact",
			},
		},
		{
			name: "nil entries slice produces valid XML",
			entries: nil,
			wantXMLHeader: true,
			wantContains: []string{
				`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := site.BuildSitemap(tt.entries)
			if err != nil {
				t.Fatalf("BuildSitemap() error = %v", err)
			}

			body := string(got)

			if tt.wantXMLHeader {
				const header = `<?xml version="1.0" encoding="UTF-8"?>`
				if !strings.HasPrefix(body, header) {
					t.Errorf("BuildSitemap() output does not start with XML header\nGot:\n%s", body)
				}
			}

			for _, loc := range tt.wantLocs {
				needle := "<loc>" + loc + "</loc>"
				if !strings.Contains(body, needle) {
					t.Errorf("BuildSitemap() output missing loc %q\nGot:\n%s", loc, body)
				}
			}

			for _, loc := range tt.wantNoLocs {
				needle := "<loc>" + loc + "</loc>"
				if strings.Contains(body, needle) {
					t.Errorf("BuildSitemap() output should not contain loc %q\nGot:\n%s", loc, body)
				}
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(body, want) {
					t.Errorf("BuildSitemap() output missing %q\nGot:\n%s", want, body)
				}
			}

			for _, noWant := range tt.wantNotContains {
				if strings.Contains(body, noWant) {
					t.Errorf("BuildSitemap() output should not contain %q\nGot:\n%s", noWant, body)
				}
			}

			// Ensure the output is valid XML in all cases.
			var set parsedURLSet
			if err := xml.Unmarshal(got, &set); err != nil {
				t.Fatalf("BuildSitemap() output is not valid XML: %v\nBody:\n%s", err, body)
			}
		})
	}
}

func TestBuildSitemapRoundTrip(t *testing.T) {
	fixedTime := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	entries := []site.Entry{
		{
			Loc:        "https://example.com/",
			LastMod:    tp(fixedTime),
			ChangeFreq: "daily",
			Priority:   fp(1.0),
		},
		{
			Loc:        "https://example.com/blog",
			LastMod:    tp(time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)),
			ChangeFreq: "weekly",
			Priority:   fp(0.7),
		},
		{
			Loc: "https://example.com/about",
			// nil LastMod and nil Priority
		},
	}

	data, err := site.BuildSitemap(entries)
	if err != nil {
		t.Fatalf("BuildSitemap() error = %v", err)
	}

	// Strip the XML declaration before unmarshalling (xml.Unmarshal handles it
	// but some versions may be strict; using the raw bytes is fine).
	var set parsedURLSet
	if err := xml.Unmarshal(data, &set); err != nil {
		t.Fatalf("xml.Unmarshal error = %v\nBody:\n%s", err, data)
	}

	if got, want := set.XMLNS, "http://www.sitemaps.org/schemas/sitemap/0.9"; got != want {
		t.Errorf("XMLNS = %q, want %q", got, want)
	}

	if len(set.URLs) != len(entries) {
		t.Fatalf("len(URLs) = %d, want %d", len(set.URLs), len(entries))
	}

	cases := []struct {
		idx            int
		wantLoc        string
		wantLastMod    string
		wantChangeFreq string
		wantPriority   string
	}{
		{0, "https://example.com/", "2024-06-01", "daily", "1.0"},
		{1, "https://example.com/blog", "2024-01-10", "weekly", "0.7"},
		{2, "https://example.com/about", "", "", ""},
	}

	for _, c := range cases {
		u := set.URLs[c.idx]
		if u.Loc != c.wantLoc {
			t.Errorf("URL[%d].Loc = %q, want %q", c.idx, u.Loc, c.wantLoc)
		}
		if u.LastMod != c.wantLastMod {
			t.Errorf("URL[%d].LastMod = %q, want %q", c.idx, u.LastMod, c.wantLastMod)
		}
		if u.ChangeFreq != c.wantChangeFreq {
			t.Errorf("URL[%d].ChangeFreq = %q, want %q", c.idx, u.ChangeFreq, c.wantChangeFreq)
		}
		if u.Priority != c.wantPriority {
			t.Errorf("URL[%d].Priority = %q, want %q", c.idx, u.Priority, c.wantPriority)
		}
	}
}

func TestBuildSitemapXMLHeader(t *testing.T) {
	// Verify XML declaration is present in all output, even for empty sitemap.
	data, err := site.BuildSitemap(nil)
	if err != nil {
		t.Fatalf("BuildSitemap(nil) error = %v", err)
	}
	const header = `<?xml version="1.0" encoding="UTF-8"?>`
	if !strings.HasPrefix(string(data), header) {
		t.Errorf("output does not begin with XML declaration\nGot: %s", data)
	}
}

func TestBuildSitemapNamespace(t *testing.T) {
	data, err := site.BuildSitemap([]site.Entry{{Loc: "https://example.com/"}})
	if err != nil {
		t.Fatalf("BuildSitemap() error = %v", err)
	}

	var set parsedURLSet
	if err := xml.Unmarshal(data, &set); err != nil {
		t.Fatalf("xml.Unmarshal error = %v", err)
	}
	if got, want := set.XMLNS, "http://www.sitemaps.org/schemas/sitemap/0.9"; got != want {
		t.Errorf("xmlns = %q, want %q", got, want)
	}
}
