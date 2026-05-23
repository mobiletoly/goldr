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

  function sameAnchor(a, b) {
    return Math.abs(a.left - b.left) < 8 && Math.abs(a.top - b.top) < 8;
  }

  function trackedBoxes() {
    var boxes = [];
    markers().forEach(function (pair) {
      var range = document.createRange();
      range.setStartAfter(pair.start);
      range.setEndBefore(pair.end);
      var rect = range.getBoundingClientRect();
      range.detach();

      if (rect.width < 1 || rect.height < 1) {
        return;
      }

      var track = 0;
      boxes.forEach(function (box) {
        if (sameAnchor(rect, box.rect)) {
          track = Math.max(track, box.track + 1);
        }
      });
      boxes.push({ rect: rect, meta: pair.meta, track: track });
    });
    return boxes;
  }

  function appendBox(root, rect, meta, track) {
    if (rect.width < 1 || rect.height < 1) {
      return;
    }

    var color = colors[meta.kind] || "#7c3aed";
    var inset = Math.min(track * 3, Math.floor(Math.min(rect.width, rect.height) / 4));
    var box = document.createElement("div");
    box.setAttribute("data-goldr-template-stack", String(track));
    box.style.cssText = [
      "position:fixed",
      "left:" + (rect.left + inset) + "px",
      "top:" + (rect.top + inset) + "px",
      "width:" + Math.max(0, rect.width - inset * 2) + "px",
      "height:" + Math.max(0, rect.height - inset * 2) + "px",
      "box-sizing:border-box",
      "border:2px solid " + color,
      "pointer-events:none"
    ].join(";");

    root.appendChild(box);
    appendLabel(root, rect, meta, inset);
  }

  function appendLabel(root, rect, meta, inset) {
    var color = colors[meta.kind] || "#7c3aed";
    var label = document.createElement("div");
    label.setAttribute("data-goldr-template-label", "1");
    label.textContent = labelText(meta);
    label.style.cssText = [
      "position:fixed",
      "left:" + (rect.left + inset - 2) + "px",
      "top:" + (rect.top + inset - 2) + "px",
      "max-width:min(520px,calc(100% - 8px),90vw)",
      "overflow:hidden",
      "text-overflow:ellipsis",
      "white-space:nowrap",
      "background:" + color,
      "color:white",
      "font:11px/15px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
      "padding:1px 4px",
      "border-radius:2px"
    ].join(";");

    root.appendChild(label);
  }

  function paddedOverlap(a, b, padding) {
    return a.left < b.right + padding &&
      a.right + padding > b.left &&
      a.top < b.bottom + padding &&
      a.bottom + padding > b.top;
  }

  function labelRectAt(rect, top) {
    return {
      left: rect.left,
      right: rect.right,
      top: top,
      bottom: top + rect.height
    };
  }

  function layoutLabels(root) {
    var placed = [];
    var labels = Array.prototype.slice.call(root.querySelectorAll("[data-goldr-template-label]"));
    labels.sort(function (a, b) {
      var aRect = a.getBoundingClientRect();
      var bRect = b.getBoundingClientRect();
      return (aRect.top - bRect.top) || (aRect.left - bRect.left);
    });

    labels.forEach(function (label) {
      var rect = label.getBoundingClientRect();
      var top = rect.top;
      var moved = true;
      while (moved) {
        moved = false;
        placed.forEach(function (placedRect) {
          if (paddedOverlap(labelRectAt(rect, top), placedRect, 4)) {
            top = Math.max(top, placedRect.bottom + 4);
            moved = true;
          }
        });
      }
      label.style.top = top + "px";
      label.setAttribute("data-goldr-template-label-row", String(Math.round(top - rect.top)));
      placed.push(labelRectAt(rect, top));
    });
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

    trackedBoxes().forEach(function (box) {
      appendBox(root, box.rect, box.meta, box.track);
    });

    if (root.childNodes.length > 0) {
      document.body.appendChild(root);
      layoutLabels(root);
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
