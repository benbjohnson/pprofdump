package main_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"

	"os"
	"testing"

	main "github.com/benbjohnson/pprofdump"
)

// Ensure the program can parse command line arguments.
func TestMain_ParseFlags(t *testing.T) {
	m := NewMain()
	if err := m.ParseFlags([]string{"http://localhost:1000"}); err != nil {
		t.Fatal(err)
	}
	if m.URL != (url.URL{Scheme: "http", Host: "localhost:1000"}) {
		t.Fatalf("unexpected url: %s", m.URL.String())
	}
}

// Ensure the program can fetch pprof from
func TestMain_Run(t *testing.T) {
	// Run server that simply echos profile paths.
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, r.URL.Path)
	}))
	defer s.Close()

	// Execute against mock server.
	m := NewMain()
	m.URL = *MustParseURL(s.URL)
	m.ProfileNames = []string{"foo", "bar"}
	if err := m.Run(); err != nil {
		t.Fatal(err)
	}

	// Verify stdout.
	gr, err := gzip.NewReader(&m.Stdout)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gr)
	for i := 0; i < len(m.ProfileNames); i++ {
		hdr, err := tr.Next()
		if err != nil {
			t.Fatal(err)
		}

		// Verify file contents.
		buf, err := ioutil.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		var exp []byte
		switch hdr.Name {
		case "foo":
			exp = []byte("/debug/pprof/foo")
		case "bar":
			exp = []byte("/debug/pprof/bar")
		default:
			t.Fatalf("unexpected tar file: %s", hdr.Name)
		}
		if !bytes.Equal(buf, exp) {
			t.Fatalf("unexpected tar file(%s): got=%s, exp=%s", hdr.Name, buf, exp)
		}
	}

	// TODO: Verify logs.
}

// Main is a test wrapper for main.Main.
type Main struct {
	*main.Main

	Stdin  bytes.Buffer
	Stdout bytes.Buffer
	Stderr bytes.Buffer
}

// NewMain returns a new instance of Main.
// If the verbose flag is set then STDOUT/STDERR are displayed on the screen.
func NewMain() *Main {
	m := &Main{Main: main.NewMain()}
	m.Main.Stdin = &m.Stdin
	m.Main.Stdout = &m.Stdout
	m.Main.Stderr = NewVerboseWriter(&m.Stderr)
	return m
}

// NewVerboseWriter returns w optionally wrapped by a MultiWriter if verbose flag is set.
func NewVerboseWriter(w io.Writer) io.Writer {
	if testing.Verbose() {
		return io.MultiWriter(os.Stderr, w)
	}
	return w
}

// MustParseURL parses rawurl into a URL. Panic on error.
func MustParseURL(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		panic(err)
	}
	return u
}
