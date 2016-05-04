package main

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
)

// DefaultPrefix is the default URL prefix.
const DefaultPrefix = "/debug/pprof"

// DefaultProfileNames are the default profiles fetched.
var DefaultProfileNames = []string{
	"profile",
	"heap",
	"block",
	"goroutine?debug=2",
	"threadcreate",
}

func main() {
	m := NewMain()

	// Parse command line flags.
	if err := m.ParseFlags(os.Args[1:]); err == flag.ErrHelp {
		fmt.Fprintln(m.Stderr, m.Usage())
		os.Exit(2)
	} else if err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}

	// Execute program.
	if err := m.Run(); err != nil {
		fmt.Fprintln(m.Stderr, err)
		os.Exit(1)
	}
}

// Main represents the main program execution.
type Main struct {
	// Base URL to call against.
	URL    url.URL
	Prefix string

	// File path to write to.
	OutputPath string

	// List of profile names to retrieve from.
	ProfileNames []string

	// Show verbose logging.
	Verbose bool

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

// NewMain returns a new instance of Main.
func NewMain() *Main {
	return &Main{
		Prefix:       DefaultPrefix,
		ProfileNames: DefaultProfileNames,

		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
}

// Usage returns the usage message.
func (m *Main) Usage() string {
	return `
profdump is a tool for retrieving all pprof profiles at once.
Profiles are then combined into a single gzipped tar archive.

Usage:
	profdump [arguments] URL > output.tar.gz

The following flags are available:

	-o PATH
	   File path to write the output to.
	   Defaults to stdout.

	-profiles NAME,NAME,NAME
	    Comma-delimited list of profiles to fetch.
	    Defaults to profile,heap,block,goroutine,threadcreate

	-v
	    Show verbose output. If not specified, only shows errors.

`[1:]
}

// ParseFlags parses the command line flags.
func (m *Main) ParseFlags(args []string) error {
	// Parse flags.
	fs := flag.NewFlagSet("profdump", flag.ContinueOnError)
	fs.SetOutput(ioutil.Discard)
	fs.StringVar(&m.OutputPath, "o", "", "")
	profiles := fs.String("profiles", strings.Join(DefaultProfileNames, ","), "")
	fs.BoolVar(&m.Verbose, "v", false, "")
	if err := fs.Parse(args); err != nil {
		return err
	}

	// Split profiles into a slice.
	m.ProfileNames = strings.Split(*profiles, ",")

	// Validate argument count.
	if fs.NArg() == 0 {
		return errors.New("URL required")
	} else if fs.NArg() > 1 {
		return errors.New("too many arguments")
	}

	// Parse first argument as a URL.
	u, err := url.Parse(fs.Arg(0))
	if err != nil {
		return errors.New("invalid URL")
	}
	m.URL = *u

	return nil
}

// Run executes the program.
func (m *Main) Run() error {
	logger := m.logger()

	// Read from separate goroutines.
	results := make(chan profileReadCloser)
	m.fetchProfiles(m.ProfileNames, results)

	// Determine output writer.
	var w io.Writer
	if m.OutputPath == "" {
		w = m.Stdout
	} else {
		f, err := os.Create(m.OutputPath)
		if err != nil {
			return err
		}
		defer f.Close()
		w = f
	}

	// Open output file.
	gw := gzip.NewWriter(w)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Write to archive.
	var successN int
	for r := range results {
		if err := m.writeTarFile(tw, r.name, r.ReadCloser); err != nil {
			logger.Printf("[%s] archive failed: %s", r.name, err)
		} else {
			successN++
		}
	}

	// Close gzip/tar writers and check for errors.
	if err := tw.Close(); err != nil {
		return err
	} else if err := gw.Close(); err != nil {
		return err
	}

	// Close writer, if applicable.
	if w, ok := w.(io.Closer); ok {
		if err := w.Close(); err != nil {
			return err
		}
	}

	// If no files were written successfully then return error.
	if successN == 0 {
		if m.OutputPath != "" {
			os.Remove(m.OutputPath)
		}
		return errors.New("no profiles written")
	}

	// Write success message, if verbose logging enabled.
	if m.Verbose {
		m.logger().Printf("%d profiles successfully written", successN)
	}

	return nil
}

// fetchProfiles reads all profiles concurrently and returns them to the results channel.
func (m *Main) fetchProfiles(names []string, results chan profileReadCloser) {
	var wg sync.WaitGroup
	for _, name := range names {
		// Log fetch start, if verbose logging enabled.
		if m.Verbose {
			segment, _ := ParseProfileName(name)
			m.logger().Printf("[%s] fetching", segment)
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			m.fetchProfile(name, results)
		}(name)
	}

	go func() {
		wg.Wait()
		close(results)
	}()
}

// fetchProfile reads a named profile and returns it on the results channel.
func (m *Main) fetchProfile(name string, results chan profileReadCloser) {
	// Split into path segment and params.
	segment, query := ParseProfileName(name)

	// Construct URL.
	u := m.URL
	u.Path = path.Join(m.Prefix, segment)
	u.RawQuery = query

	// Fetch profile over HTTP.
	resp, err := http.Get(u.String())
	if err != nil {
		m.logger().Printf("[%s] error: %s", segment, err)
		return
	} else if resp.StatusCode != http.StatusOK {
		m.logger().Printf("[%s] error: status=%d", segment, resp.StatusCode)
		resp.Body.Close()
		return
	}

	// Return profile body back to result channel.
	results <- profileReadCloser{name: name, ReadCloser: resp.Body}
}

// writeTarFile writes a profile to a tar file.
func (m *Main) writeTarFile(tw *tar.Writer, name string, r io.ReadCloser) error {

	// Read entire profile into buffer.
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	// Remove special characters from filename.
	filename := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(name, "-")

	// Write tar header.
	if err := tw.WriteHeader(&tar.Header{
		Name: filename,
		Mode: 0600,
		Size: int64(len(buf)),
	}); err != nil {
		return err
	}

	// Write body.
	if _, err := tw.Write(buf); err != nil {
		return err
	}

	// Log successful write, if verbose logging enabled.
	if m.Verbose {
		segment, _ := ParseProfileName(name)
		m.logger().Printf("[%s] OK", segment)
	}

	return nil
}

func (m *Main) logger() *log.Logger { return log.New(m.Stderr, "", log.LstdFlags) }

type profileReadCloser struct {
	io.ReadCloser
	name string
}

// ParseProfileName parses a profile name into its segment & query parts.
func ParseProfileName(name string) (segment, query string) {
	a := strings.Split(name, "?")
	if len(a) == 1 {
		return a[0], ""
	}
	return a[0], a[1]
}
