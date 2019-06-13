package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"

	"cloud.google.com/go/trace"
	"github.com/octo/retry"
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

	// sample 100% of requests, but at most 5 per second.
	policy, err := trace.NewLimitedSampler(1.0, 5.0)
	if err != nil {
		log.Fatalf("NewLimitedSampler(): %v", err)
	}
	traceClient.SetSamplingPolicy(policy)

	var h http.Handler
	h = http.HandlerFunc(handler)
	h = &retry.BudgetHandler{
		Handler: h,
		Budget: retry.Budget{
			Rate:  2.0,
			Ratio: 0.1,
		},
	}
	h = traceClient.HTTPHandler(h)

	http.HandleFunc("/_ah/health", healthCheckHandler)
	http.Handle("/", h)

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "https://github.com/octo/clang-format-gae/", http.StatusFound)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := format(ctx, r.Body, w); err != nil {
		log.Printf("contextHandler: %v", err)

		select {
		case <-ctx.Done():
			http.Error(w, "request timed out", http.StatusRequestTimeout)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
}

func format(ctx context.Context, in io.Reader, out io.Writer) error {
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
