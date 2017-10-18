package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os/exec"
)

const (
	clangFormat = "/usr/bin/clang-format"
)

func main() {
	http.HandleFunc("/_ah/health", healthCheckHandler)
	http.HandleFunc("/", handler)

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	if err := contextHandler(r.Context(), w, r); err != nil {
		log.Printf("contextHandler: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func contextHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if r.Method != "POST" {
		http.Redirect(w, r, "https://github.com/octo/clang-format-gae/", http.StatusFound)
		return nil
	}

	formatted, err := format(ctx, r.Body)
	if err != nil {
		return err
	}

	_, err = formatted.WriteTo(w)
	return err
}

func format(ctx context.Context, in io.Reader) (*bytes.Buffer, error) {
	cmd := exec.CommandContext(ctx, clangFormat, "-style=LLVM")
	cmd.Stdin = in

	out := &bytes.Buffer{}
	cmd.Stdout = out

	errbuf := &bytes.Buffer{}
	cmd.Stderr = errbuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("clang-format: %v\nSTDERR: %s", err, errbuf)
	}

	return out, nil
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}
