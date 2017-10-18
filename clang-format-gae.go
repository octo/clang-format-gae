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
	if r.Method != "POST" {
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

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("clang-format: %v\nSTDERR: %s", err, errbuf)
	}

	return nil
}

func healthCheckHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintln(w, "ok")
}
