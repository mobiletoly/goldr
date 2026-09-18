// Copyright 2026 Toly Pochkin
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/mobiletoly/goldr"
	"github.com/mobiletoly/goldr/content"
	"github.com/mobiletoly/goldr/examples/content_pages/app/routes"
	"github.com/mobiletoly/goldr/examples/content_pages/assets"
)

const defaultAddr = "127.0.0.1:8080"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	if err := run(ctx, os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "goldr content pages example: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	flags := flag.NewFlagSet("content-pages", flag.ContinueOnError)
	flags.SetOutput(stderr)
	addr := flags.String("addr", defaultAddr, "HTTP listen address")
	dev := flags.Bool("dev", false, "enable template inspection comments")
	checkContent := flags.Bool("check-content", false, "validate content and exit")
	if err := flags.Parse(args); err != nil {
		return err
	}

	root, err := os.OpenRoot("content")
	if err != nil {
		return fmt.Errorf("open content root: %w", err)
	}
	defer root.Close()

	if *checkContent {
		if err := content.Check(root.FS()); err != nil {
			return fmt.Errorf("check content: %w", err)
		}
		_, err := fmt.Fprintln(stdout, "content is valid")
		return err
	}

	handler, err := exampleHandler(root.FS(), stderr, *dev)
	if err != nil {
		return err
	}

	var listenConfig net.ListenConfig
	listener, err := listenConfig.Listen(ctx, "tcp", *addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", *addr, err)
	}

	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	if _, err := fmt.Fprintf(stdout, "goldr content pages example listening on http://%s\n", listener.Addr().String()); err != nil {
		return fmt.Errorf("write launch URL: %w", err)
	}
	if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func exampleHandler(files fs.FS, logOutput io.Writer, dev bool) (http.Handler, error) {
	pages, err := content.New(content.Config{FS: files})
	if err != nil {
		return nil, fmt.Errorf("load content: %w", err)
	}
	logger := log.New(logOutput, "content-pages: ", 0)
	inspection := goldr.TemplateInspectionOff
	if dev {
		inspection = goldr.TemplateInspectionComments
	}

	routesHandler := routes.HandlerWithOptions(routes.HandlerOptions{
		ErrorHandlers: routes.ErrorHandlers{
			RouteNotFound: routes.RouteNotFound,
			RouteError: func(r *http.Request, err error) goldr.RouteResponse {
				logger.Printf("%s: %v", r.URL.EscapedPath(), err)
				return routes.InternalError()
			},
		},
		Fallback:           pages.Resolve,
		TemplateInspection: inspection,
	})

	mux := http.NewServeMux()
	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assets.FS()))))
	mux.Handle("/", outerContext(noStore(routesHandler)))
	return mux, nil
}

func outerContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, routes.WithOuterContext(r, "outer middleware"))
	})
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
