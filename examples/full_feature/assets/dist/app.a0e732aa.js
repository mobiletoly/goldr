// Mark that the app-owned JavaScript loaded; tests use this as a cheap asset check.
document.documentElement.dataset.goldrJs = "ready";

for (const element of document.querySelectorAll("[data-js-enhance]")) {
  element.dataset.jsEnhanced = "true";
}
