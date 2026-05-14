document.documentElement.dataset.goldrJs = "ready";

for (const element of document.querySelectorAll("[data-js-enhance]")) {
  element.dataset.jsEnhanced = "true";
}
