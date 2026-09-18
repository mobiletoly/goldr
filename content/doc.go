// Package content loads trusted HTML and Markdown pages from an fs.FS.
//
// Applications configure Pages once, then assign Pages.Resolve to the
// generated routes.HandlerOptions Fallback field. Goldr keeps application
// routes first and renders resolved pages through the application's root
// layout. HTML passes through without validation or sanitization, and Markdown
// permits raw HTML and potentially dangerous destinations. Content authors
// must be trusted like application template authors; public or otherwise
// untrusted content is unsupported.
package content
