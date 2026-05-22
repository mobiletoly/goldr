(function () {
  "use strict";

  var overlayAttribute = "data-goldr-template-inspector";
  var colors = {
    layout: "#2563eb",
    page: "#16a34a",
    fragment: "#f97316"
  };

  function parseMeta(text) {
    var meta = {};
    text.replace(/([a-z]+)=([^ ]+)/g, function (_, key, value) {
      meta[key] = value;
      return "";
    });
    return meta;
  }

  function markers() {
    var starts = [];
    var pairs = [];
    var walker = document.createTreeWalker(document, NodeFilter.SHOW_COMMENT);
    var node = walker.nextNode();
    while (node) {
      var text = node.nodeValue.trim();
      if (text.indexOf("goldr:start ") === 0) {
        starts.push({ node: node, meta: parseMeta(text) });
      } else if (text.indexOf("goldr:end ") === 0) {
        var id = parseMeta(text).id;
        for (var index = starts.length - 1; index >= 0; index--) {
          if (starts[index].meta.id === id) {
            pairs.push({ start: starts[index].node, end: node, meta: starts[index].meta });
            starts.splice(index, 1);
            break;
          }
        }
      }
      node = walker.nextNode();
    }
    return pairs;
  }

  function removeOverlay() {
    var existing = document.querySelector("[" + overlayAttribute + "]");
    if (existing) {
      existing.remove();
    }
  }

  function labelText(meta) {
    return (meta.kind || "template") + ": " + (meta.source || meta.go || meta.route || "unknown");
  }

  function colorWithAlpha(color, alpha) {
    var match = /^#([0-9a-f]{6})$/i.exec(color);
    if (!match) {
      return color;
    }
    var value = match[1];
    return "rgba(" + parseInt(value.slice(0, 2), 16) + "," +
      parseInt(value.slice(2, 4), 16) + "," +
      parseInt(value.slice(4, 6), 16) + "," + alpha + ")";
  }

  function appendBox(root, rect, meta) {
    if (rect.width < 1 || rect.height < 1) {
      return;
    }

    var color = colors[meta.kind] || "#7c3aed";
    var box = document.createElement("div");
    box.style.cssText = [
      "position:fixed",
      "left:" + rect.left + "px",
      "top:" + rect.top + "px",
      "width:" + rect.width + "px",
      "height:" + rect.height + "px",
      "box-sizing:border-box",
      "border:2px solid " + color,
      "pointer-events:none"
    ].join(";");

    var label = document.createElement("div");
    label.textContent = labelText(meta);
    label.style.cssText = [
      "position:absolute",
      "left:-2px",
      "top:-2px",
      "max-width:min(520px,calc(100% - 8px),90vw)",
      "overflow:hidden",
      "text-overflow:ellipsis",
      "white-space:nowrap",
      "background:" + colorWithAlpha(color, 0.7),
      "color:white",
      "font:11px/15px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
      "padding:1px 4px",
      "border-radius:2px"
    ].join(";");

    box.appendChild(label);
    root.appendChild(box);
  }

  function draw() {
    removeOverlay();

    var root = document.createElement("div");
    root.setAttribute(overlayAttribute, "1");
    root.setAttribute("aria-hidden", "true");
    root.style.cssText = [
      "position:fixed",
      "inset:0",
      "pointer-events:none",
      "z-index:2147483647"
    ].join(";");

    markers().forEach(function (pair) {
      var range = document.createRange();
      range.setStartAfter(pair.start);
      range.setEndBefore(pair.end);
      appendBox(root, range.getBoundingClientRect(), pair.meta);
      range.detach();
    });

    if (root.childNodes.length > 0) {
      document.body.appendChild(root);
    }
  }

  function scheduleDraw() {
    window.clearTimeout(scheduleDraw.timer);
    scheduleDraw.timer = window.setTimeout(draw, 50);
  }

  function isOverlayNode(node) {
    if (!node || node.nodeType !== 1) {
      return false;
    }
    if (node.hasAttribute(overlayAttribute)) {
      return true;
    }
    return Boolean(node.closest("[" + overlayAttribute + "]"));
  }

  function isOverlayMutation(mutation) {
    if (isOverlayNode(mutation.target)) {
      return true;
    }

    var nodes = Array.prototype.concat.call(
      Array.prototype.slice.call(mutation.addedNodes),
      Array.prototype.slice.call(mutation.removedNodes)
    );
    return nodes.length > 0 && nodes.every(isOverlayNode);
  }

  function watchDOMChanges() {
    if (!window.MutationObserver || !document.body) {
      return;
    }

    var observer = new MutationObserver(function (mutations) {
      if (mutations.every(isOverlayMutation)) {
        return;
      }
      scheduleDraw();
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });
  }

  window.addEventListener("load", scheduleDraw);
  window.addEventListener("resize", scheduleDraw);
  window.addEventListener("scroll", scheduleDraw, true);
  document.addEventListener("htmx:afterSwap", scheduleDraw);
  document.addEventListener("htmx:afterSettle", scheduleDraw);
  watchDOMChanges();
  scheduleDraw();
})();
