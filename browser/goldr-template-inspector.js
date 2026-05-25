(function () {
  "use strict";

  var overlayAttribute = "data-goldr-template-inspector";
  var controlAttribute = "data-goldr-template-inspector-control";
  var modeAttribute = "data-goldr-template-inspector-mode";
  var nextAttribute = "data-goldr-template-inspector-next";
  var storageKey = "goldr.templateInspector.mode";
  var overlayZIndex = "2147483646";
  var controlZIndex = "2147483647";
  var mode = storedMode();
  var selectedIndex = null;
  var colors = {
    layout: "#2563eb",
    page: "#16a34a",
    fragment: "#f97316"
  };

  function storedMode() {
    try {
      if (window.localStorage && window.localStorage.getItem(storageKey) === "off") {
        return "off";
      }
    } catch (_) {
      return "all";
    }
    return "all";
  }

  function persistMode(value) {
    try {
      if (window.localStorage) {
        window.localStorage.setItem(storageKey, value === "off" ? "off" : "all");
      }
    } catch (_) {
      return;
    }
  }

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
    var sequence = 0;
    var walker = document.createTreeWalker(document, NodeFilter.SHOW_COMMENT);
    var node = walker.nextNode();
    while (node) {
      var text = node.nodeValue.trim();
      if (text.indexOf("goldr:start ") === 0) {
        starts.push({ node: node, meta: parseMeta(text), depth: starts.length, sequence: sequence });
        sequence += 1;
      } else if (text.indexOf("goldr:end ") === 0) {
        var id = parseMeta(text).id;
        for (var index = starts.length - 1; index >= 0; index--) {
          if (starts[index].meta.id === id) {
            pairs.push({
              start: starts[index].node,
              end: node,
              meta: starts[index].meta,
              depth: starts[index].depth,
              sequence: starts[index].sequence
            });
            starts.splice(index, 1);
            break;
          }
        }
      }
      node = walker.nextNode();
    }
    return pairs;
  }

  function boxOrder(a, b) {
    return (a.depth - b.depth) || (a.sequence - b.sequence);
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

  function sourcePath(meta) {
    return meta.source || meta.go || meta.route || "unknown";
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

      boxes.push({
        rect: rect,
        meta: pair.meta,
        depth: pair.depth,
        sequence: pair.sequence,
        track: 0
      });
    });

    boxes.sort(boxOrder);
    boxes.forEach(function (box, index) {
      var track = 0;
      boxes.slice(0, index).forEach(function (previous) {
        if (sameAnchor(box.rect, previous.rect)) {
          track = Math.max(track, previous.track + 1);
        }
      });
      box.track = track;
    });
    return boxes;
  }

  function activeBoxes(boxes) {
    if (mode === "off") {
      return [];
    }
    if (selectedIndex === null) {
      return boxes;
    }
    if (boxes.length === 0) {
      selectedIndex = null;
      return [];
    }
    if (selectedIndex >= boxes.length) {
      selectedIndex = 0;
    }
    return [boxes[selectedIndex]];
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
    label.style.cssText = [
      "position:fixed",
      "left:" + (rect.left + inset - 2) + "px",
      "top:" + (rect.top + inset - 2) + "px",
      "max-width:min(520px,calc(100% - 8px),90vw)",
      "display:flex",
      "align-items:center",
      "gap:4px",
      "background:" + color,
      "color:white",
      "font:11px/15px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
      "padding:1px 4px",
      "border-radius:2px",
      "pointer-events:auto"
    ].join(";");

    var text = document.createElement("span");
    text.textContent = labelText(meta);
    text.style.cssText = [
      "min-width:0",
      "overflow:hidden",
      "text-overflow:ellipsis",
      "white-space:nowrap"
    ].join(";");
    label.appendChild(text);
    appendCopyButton(label, sourcePath(meta));
    root.appendChild(label);
  }

  function appendCopyButton(label, path) {
    var button = document.createElement("button");
    button.type = "button";
    button.textContent = "\u29c9";
    button.title = "Copy source path to clipboard";
    button.setAttribute("aria-label", "Copy source path to clipboard");
    button.setAttribute("data-goldr-template-copy", "1");
    button.style.cssText = [
      "appearance:none",
      "flex:0 0 auto",
      "border:0",
      "border-radius:2px",
      "background:rgba(255,255,255,.18)",
      "color:inherit",
      "font:11px/13px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
      "padding:0 3px",
      "cursor:pointer"
    ].join(";");
    button.addEventListener("click", function (event) {
      event.preventDefault();
      event.stopPropagation();
      copySourcePath(path, button);
    });
    label.appendChild(button);
  }

  function copySourcePath(path, button) {
    function copied() {
      button.title = "Copied source path";
      window.setTimeout(function () {
        button.title = "Copy source path to clipboard";
      }, 1200);
    }

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(path).then(copied, function () {
        fallbackCopySourcePath(path);
        copied();
      });
      return;
    }

    fallbackCopySourcePath(path);
    copied();
  }

  function fallbackCopySourcePath(path) {
    var input = document.createElement("textarea");
    input.value = path;
    input.setAttribute("readonly", "readonly");
    input.style.cssText = [
      "position:fixed",
      "left:-9999px",
      "top:0"
    ].join(";");
    document.body.appendChild(input);
    input.select();
    try {
      document.execCommand("copy");
    } catch (_) {
      return;
    } finally {
      input.remove();
    }
  }

  function buttonStyle(active, disabled) {
    if (disabled) {
      return [
        "appearance:none",
        "border:1px solid #9ca3af",
        "border-radius:3px",
        "background:#f3f4f6",
        "color:#6b7280",
        "font:12px/16px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
        "padding:2px 7px",
        "cursor:not-allowed",
        "opacity:.72"
      ].join(";");
    }
    return [
      "appearance:none",
      "border:1px solid #111827",
      "border-radius:3px",
      "background:" + (active ? "#111827" : "#ffffff"),
      "color:" + (active ? "#ffffff" : "#111827"),
      "font:12px/16px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
      "padding:2px 7px",
      "cursor:pointer"
    ].join(";");
  }

  function appendModeButton(root, label, value) {
    var button = document.createElement("button");
    button.type = "button";
    button.textContent = label;
    button.title = "Show " + label.toLowerCase() + " template inspector overlays";
    button.setAttribute(modeAttribute, value);
    button.setAttribute("aria-pressed", mode === value && selectedIndex === null ? "true" : "false");
    button.style.cssText = buttonStyle(mode === value && selectedIndex === null, false);
    button.addEventListener("click", function () {
      mode = value;
      selectedIndex = null;
      persistMode(value);
      draw();
    });
    root.appendChild(button);
  }

  function appendNextButton(root) {
    var button = document.createElement("button");
    button.type = "button";
    button.textContent = "Next";
    button.title = "Show the next template inspector overlay";
    button.setAttribute(nextAttribute, "1");
    var disabled = mode === "off";
    button.disabled = disabled;
    button.setAttribute("aria-pressed", mode !== "off" && selectedIndex !== null ? "true" : "false");
    button.style.cssText = buttonStyle(mode !== "off" && selectedIndex !== null, disabled) + ";margin-left:10px";
    button.addEventListener("click", function () {
      var boxes = trackedBoxes();
      if (boxes.length > 0) {
        if (selectedIndex === null) {
          selectedIndex = 0;
        } else {
          selectedIndex = (selectedIndex + 1) % boxes.length;
        }
        mode = "all";
        persistMode("all");
      }
      draw();
    });
    root.appendChild(button);
  }

  function drawControl() {
    if (!document.body) {
      return;
    }

    var root = document.querySelector("[" + controlAttribute + "]");
    if (!root) {
      root = document.createElement("div");
      root.setAttribute(controlAttribute, "1");
      root.setAttribute("role", "group");
      root.setAttribute("aria-label", "Goldr template inspector controls");
      document.body.appendChild(root);
    }

    root.style.cssText = [
      "position:fixed",
      "right:12px",
      "bottom:12px",
      "z-index:" + controlZIndex,
      "display:flex",
      "align-items:center",
      "gap:4px",
      "padding:4px",
      "border:1px solid rgba(17,24,39,.22)",
      "border-radius:4px",
      "background:rgba(255,255,255,.94)",
      "box-shadow:0 2px 10px rgba(17,24,39,.16)",
      "pointer-events:auto"
    ].join(";");

    root.textContent = "";
    appendModeButton(root, "All", "all");
    appendModeButton(root, "Off", "off");
    appendNextButton(root);
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
    drawControl();

    var boxes = trackedBoxes();
    var boxesToDraw = activeBoxes(boxes);
    if (boxesToDraw.length === 0) {
      return;
    }

    var root = document.createElement("div");
    root.setAttribute(overlayAttribute, "1");
    root.style.cssText = [
      "position:fixed",
      "inset:0",
      "pointer-events:none",
      "z-index:" + overlayZIndex
    ].join(";");

    boxesToDraw.forEach(function (box) {
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

  function isInspectorNode(node) {
    if (!node || node.nodeType !== 1) {
      return false;
    }
    if (node.hasAttribute(overlayAttribute) || node.hasAttribute(controlAttribute)) {
      return true;
    }
    return Boolean(node.closest("[" + overlayAttribute + "],[" + controlAttribute + "]"));
  }

  function isInspectorMutation(mutation) {
    if (isInspectorNode(mutation.target)) {
      return true;
    }

    var nodes = Array.prototype.concat.call(
      Array.prototype.slice.call(mutation.addedNodes),
      Array.prototype.slice.call(mutation.removedNodes)
    );
    return nodes.length > 0 && nodes.every(isInspectorNode);
  }

  function watchDOMChanges() {
    if (!window.MutationObserver || !document.body) {
      return;
    }

    var observer = new MutationObserver(function (mutations) {
      if (mutations.every(isInspectorMutation)) {
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
