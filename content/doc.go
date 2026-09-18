// Package content loads trusted HTML and Markdown pages from an fs.FS.
//
// Applications configure Pages once, then assign Pages.Resolve to the
// generated routes.HandlerOptions AdditionalPageSource field. Goldr keeps
// generated routes first. On an ordinary miss, generated routing applies the
// matching static route-tree middleware and layouts before resolving and
// rendering a page. HTML passes through without validation or sanitization, and Markdown
// permits raw HTML and potentially dangerous destinations. Content authors
// must be trusted like application template authors; public or otherwise
// untrusted content is unsupported.
package content
