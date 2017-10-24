package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
	"time"

	"cloud.google.com/go/trace"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const (
	clangFormat = "/usr/bin/clang-format"
)

func main() {
	ctx := context.Background()

	creds, err := google.FindDefaultCredentials(ctx, trace.ScopeTraceAppend)
	if err != nil {
		log.Fatalf("FindDefaultCredentials(): %v", err)
	}

	traceClient, err := trace.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.TokenSource))
	if err != nil {
		log.Fatalf("trace.NewClient(): %v", err)
	}

	// sample 100% of requests, but at most 1 per second.
	policy, err := trace.NewLimitedSampler(1.0, 1.0)
	if err != nil {
		log.Fatalf("NewLimitedSampler(): %v", err)
	}
	traceClient.SetSamplingPolicy(policy)

	http.HandleFunc("/_ah/health", healthCheckHandler)
	http.Handle("/", traceClient.HTTPHandler(http.HandlerFunc(handler)))

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "https://github.com/octo/clang-format-gae/", http.StatusFound)
		return
	}

	if err := format(r.Context(), r.Body, w); err != nil {
		log.Printf("contextHandler: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func format(ctx context.Context, in io.Reader, out io.Writer) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, clangFormat, "-style=LLVM")
	cmd.Stdin = in
	cmd.Stdout = out

	errbuf := &bytes.Buffer{}
	cmd.Stderr = errbuf

	span := trace.FromContext(ctx).NewChild(clangFormat)
	defer span.Finish()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clang-format: %v\nSTDERR: %s", err, errbuf)
	}

	return nil
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}
