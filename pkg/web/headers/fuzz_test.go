package headers

import "testing"

func FuzzParseAccept(f *testing.F) {
	f.Add("text/html")
	f.Add("text/html, application/json;q=0.9, */*;q=0.8")
	f.Add("*/*")
	f.Add("")
	f.Add("a/b;q=0;x=y, c/d;q=1.0")
	f.Fuzz(func(t *testing.T, input string) {
		a, err := ParseAccept(input)
		if err == nil {
			_ = a.String()
		}
	})
}

func FuzzParseAcceptEncoding(f *testing.F) {
	f.Add("gzip")
	f.Add("br;q=1.0, gzip;q=0.9, deflate;q=0.8")
	f.Add("identity;q=0")
	f.Add("*;q=0")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		a, err := ParseAcceptEncoding(input)
		if err == nil {
			_ = a.String()
		}
	})
}

func FuzzParseAcceptLanguage(f *testing.F) {
	f.Add("en-US")
	f.Add("en-US, en;q=0.9, fr;q=0.8")
	f.Add("*")
	f.Add("")
	f.Add("zh-CN;q=0.5, zh;q=0.3")
	f.Fuzz(func(t *testing.T, input string) {
		a, err := ParseAcceptLanguage(input)
		if err == nil {
			_ = a.String()
		}
	})
}

func FuzzParseCacheControl(f *testing.F) {
	f.Add("no-cache, no-store")
	f.Add("public, max-age=3600")
	f.Add("private, max-age=0, no-transform")
	f.Add("s-maxage=86400, stale-while-revalidate=60")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		c, err := ParseCacheControl(input)
		if err == nil {
			_ = c.String()
		}
	})
}

func FuzzParseContentDisposition(f *testing.F) {
	f.Add(`attachment; filename="foo.txt"`)
	f.Add(`attachment; filename*=UTF-8''foo%20bar.txt`)
	f.Add(`form-data; name="file"; filename="upload.png"`)
	f.Add("inline")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		c, err := ParseContentDisposition(input)
		if err == nil {
			_ = c.String()
			_ = c.PreferredFilename()
		}
	})
}

func FuzzParseContentRange(f *testing.F) {
	f.Add("bytes 0-499/1234")
	f.Add("bytes */1234")
	f.Add("bytes 0-499/*")
	f.Add("")
	f.Add("bytes 100-200/500")
	f.Fuzz(func(t *testing.T, input string) {
		c, err := ParseContentRange(input)
		if err == nil {
			_ = c.String()
		}
	})
}

func FuzzParseContentType(f *testing.F) {
	f.Add("text/html; charset=utf-8")
	f.Add("multipart/form-data; boundary=--boundary")
	f.Add("application/json")
	f.Add("")
	f.Add(`text/plain; charset="us-ascii"`)
	f.Fuzz(func(t *testing.T, input string) {
		c, err := ParseContentType(input)
		if err == nil {
			_ = c.String()
		}
	})
}

func FuzzParseCookieList(f *testing.F) {
	f.Add("session=abc123")
	f.Add("a=1; b=2; c=3")
	f.Add("__Host-session=abc; __Secure-token=xyz")
	f.Add("")
	f.Add("key=; other=val")
	f.Fuzz(func(t *testing.T, input string) {
		c := ParseCookieList(input)
		_ = c.String()
	})
}

func FuzzParseIfMatch(f *testing.F) {
	f.Add(`"abc123"`)
	f.Add(`"abc", "def"`)
	f.Add("*")
	f.Add(`W/"abc"`)
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		m, err := ParseIfMatch(input)
		if err == nil {
			_ = m.String()
			_ = m.Matches("test")
		}
	})
}

func FuzzParseIfNoneMatch(f *testing.F) {
	f.Add(`"abc123"`)
	f.Add(`W/"abc"`)
	f.Add(`"a", W/"b"`)
	f.Add("*")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		m, err := ParseIfNoneMatch(input)
		if err == nil {
			_ = m.String()
			_ = m.Matches("test", false)
			_ = m.Matches("test", true)
		}
	})
}

func FuzzParseIfRange(f *testing.F) {
	f.Add(`"abc123"`)
	f.Add("Mon, 02 Jan 2006 15:04:05 GMT")
	f.Add(`W/"abc"`)
	f.Add("")
	f.Add("Sat, 15 Jun 2024 12:00:00 GMT")
	f.Fuzz(func(t *testing.T, input string) {
		r, err := ParseIfRange(input)
		if err == nil {
			_ = r.String()
		}
	})
}

func FuzzParseRange(f *testing.F) {
	f.Add("bytes=0-499")
	f.Add("bytes=-500")
	f.Add("bytes=500-")
	f.Add("bytes=0-499, 600-999")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		r, err := ParseRange(input)
		if err == nil {
			_ = r.String()
			_ = r.CanSatisfy(1000)
		}
	})
}

func FuzzParseSetCookie(f *testing.F) {
	f.Add("session=abc123")
	f.Add("id=xyz; Domain=example.com; Path=/; HttpOnly; Secure; SameSite=Strict")
	f.Add("token=abc; Max-Age=3600")
	f.Add("key=val; Partitioned")
	f.Add("")
	f.Fuzz(func(t *testing.T, input string) {
		sc, err := ParseSetCookie(input)
		if err == nil {
			_ = sc.String()
		}
	})
}

func FuzzParseVary(f *testing.F) {
	f.Add("*")
	f.Add("Accept")
	f.Add("Accept, Accept-Encoding, Accept-Language")
	f.Add("")
	f.Add("accept, ACCEPT")
	f.Fuzz(func(t *testing.T, input string) {
		v := ParseVary(input)
		_ = v.String()
		_ = v.Has("Accept")
	})
}
