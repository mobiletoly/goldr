// Mark that the app-owned JavaScript loaded; tests use this as a cheap asset check.
document.documentElement.dataset.goldrJs = "ready";

if (window.htmx) {
  window.htmx.config.responseHandling = [
    { code: "204", swap: false },
    { code: "[23]..", swap: true },
    { code: "422", swap: true },
    { code: "[45]..", swap: false, error: true },
    { code: "...", swap: false }
  ];
}

for (const element of document.querySelectorAll("[data-js-enhance]")) {
  element.dataset.jsEnhanced = "true";
}
