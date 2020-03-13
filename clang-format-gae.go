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

	"contrib.go.opencensus.io/exporter/stackdriver"
	"contrib.go.opencensus.io/exporter/stackdriver/propagation"
	"github.com/octo/retry"
	"go.opencensus.io/plugin/ochttp"
	"go.opencensus.io/trace"
)

const (
	clangFormat = "/usr/bin/clang-format"
)

func main() {
	// Create and register a OpenCensus Stackdriver Trace exporter.
	exporter, err := stackdriver.NewExporter(stackdriver.Options{
		ProjectID: os.Getenv("GOOGLE_CLOUD_PROJECT"),
	})
	if err != nil {
		log.Fatalf("stackdriver.NewExporter(): %v", err)
	}
	trace.RegisterExporter(exporter)
	trace.ApplyConfig(trace.Config{
		DefaultSampler: trace.AlwaysSample(),
	})

	var h http.Handler
	h = &ochttp.Handler{
		Propagation:      &propagation.HTTPFormat{},
		Handler:          http.HandlerFunc(handler),
		IsPublicEndpoint: true,
	}

	h = &retry.BudgetHandler{
		Handler: h,
		Budget: retry.Budget{
			Rate:  2.0,
			Ratio: 0.1,
		},
	}

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

	ctx, span := trace.StartSpan(ctx, clangFormat)
	defer span.End()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clang-format: %v\nSTDERR: %s", err, errbuf)
	}

	return nil
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}
