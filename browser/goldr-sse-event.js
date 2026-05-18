(function () {
  if (!window.htmx || !window.htmx.registerExtension) {
    return;
  }

  window.htmx.registerExtension("goldr-sse-event", {
    htmx_before_sse_message: function (element, detail) {
      if (!element || !detail || !detail.message) {
        return;
      }

      var eventName = detail.message.event;
      if (!eventName) {
        return;
      }

      var swapEvent = element.getAttribute("goldr-sse-event");
      if (!swapEvent) {
        return;
      }

      if (eventName === swapEvent.trim()) {
        detail.message.event = "";
      }
    }
  });
})();
