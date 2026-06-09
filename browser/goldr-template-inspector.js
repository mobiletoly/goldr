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
  var expandedKey = null;
  var colors = {
    layout: "#2563eb",
    page: "#16a34a",
    fragment: "#f97316",
    component: "#7c3aed"
  };

  // State.

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

  // Small DOM helpers.

  function css(parts) {
    return parts.join(";");
  }

  function setStyle(element, parts) {
    element.style.cssText = css(parts);
  }

  function badgeButtonStyle() {
    return [
      "appearance:none",
      "flex:0 0 auto",
      "border:0",
      "border-radius:2px",
      "background:rgba(255,255,255,.18)",
      "color:inherit",
      "font:11px/13px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
      "padding:0 3px",
      "cursor:pointer"
    ];
  }

  function badgeButton(text, title, attribute) {
    var button = document.createElement("button");
    button.type = "button";
    button.textContent = text;
    button.title = title;
    button.setAttribute("aria-label", title);
    if (attribute) {
      button.setAttribute(attribute, "1");
    }
    setStyle(button, badgeButtonStyle());
    return button;
  }

  function styledTextStyle(role) {
    if (role === "primary" || role === "kind") {
      return [
        "flex:0 0 auto",
        "font-weight:700",
        "white-space:nowrap"
      ];
    }
    if (role === "component-label") {
      return [
        "flex:0 0 auto",
        "white-space:nowrap"
      ];
    }
    if (role === "path" || role === "chain-path") {
      var pathStyle = [
        role === "chain-path" ? "flex:1 1 auto" : "flex:0 0 auto",
        role === "chain-path" ? "min-width:0" : "white-space:nowrap",
        "font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace",
        "font-size:10px",
        "opacity:.95",
        "background:rgba(255,255,255,.16)",
        "border-radius:2px",
        "padding:0 3px",
        role === "chain-path" ? "white-space:normal" : "",
        role === "chain-path" ? "overflow-wrap:anywhere" : ""
      ];
      return pathStyle.filter(Boolean);
    }
    return [
      "flex:0 0 auto",
      "opacity:.7",
      "padding:0 1px"
    ];
  }

  // Marker model.

  function parseMeta(text) {
    var meta = {};
    text.replace(/([a-z]+)=([^ ]+)/g, function (_, key, value) {
      meta[key] = decodeMetaValue(value);
      return "";
    });
    return meta;
  }

  function decodeMetaValue(value) {
    return value
      .replace(/%2D/gi, "-")
      .replace(/%3E/gi, ">")
      .replace(/%20/gi, " ")
      .replace(/%25/gi, "%")
      .replace(/&gt;/g, ">")
      .replace(/- -/g, "--");
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
        starts.push({
          node: node,
          meta: parseMeta(text),
          parents: starts.map(function (start) { return start.meta; }),
          depth: starts.length,
          sequence: sequence
        });
        sequence += 1;
      } else if (text.indexOf("goldr:end ") === 0) {
        var id = parseMeta(text).id;
        for (var index = starts.length - 1; index >= 0; index--) {
          if (starts[index].meta.id === id) {
            pairs.push({
              start: starts[index].node,
              end: node,
              meta: starts[index].meta,
              parents: starts[index].parents,
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
    Array.prototype.slice.call(document.querySelectorAll("[" + overlayAttribute + "]")).forEach(function (existing) {
      existing.remove();
    });
  }

  // Inspector view model.

  function unitText(meta) {
    if (!meta) {
      return "";
    }
    if (meta.kind === "component") {
      return meta.label || "component";
    }
    if (meta.route) {
      return (meta.kind || "template") + " " + meta.route;
    }
    return (meta.kind || "template") + ": " + (sourcePathFromMeta(meta) || "unknown");
  }

  function renderContext(box) {
    for (var index = box.parents.length - 1; index >= 0; index--) {
      if (box.parents[index].kind !== "component") {
        return box.parents[index];
      }
    }
    return null;
  }

  function badgeParts(box) {
    var meta = box.meta;
    if (meta.kind === "component" && meta.label) {
      var context = renderContext(box);
      var contextPath = sourcePathFromMeta(context);
      if (contextPath) {
        return { kind: "component", primary: meta.label, secondary: contextPath };
      }
      if (context) {
        return { kind: "component", primary: meta.label, secondary: unitText(context) };
      }
      return { kind: "component", primary: meta.label, secondary: "" };
    }
    return { kind: "", primary: meta.kind || "template", secondary: sourcePathFromMeta(meta) || meta.route || "unknown" };
  }

  function sourcePathFromMeta(meta) {
    if (!meta) {
      return "";
    }
    return meta.source || meta.go || "";
  }

  function sourcePath(box) {
    if (box.meta.kind === "component") {
      return sourcePathFromMeta(renderContext(box));
    }
    return sourcePathFromMeta(box.meta);
  }

  function renderChain(box) {
    var chain = box.parents.slice(0);
    if (box.meta.kind !== "component") {
      chain.push(box.meta);
    }
    chain = chain.filter(function (meta) { return meta.kind !== "component" || meta.label; });
    if (chain.length <= 1) {
      return [];
    }
    return chain;
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
        key: (pair.meta.id || "marker") + ":" + pair.sequence,
        rect: rect,
        meta: pair.meta,
        parents: pair.parents,
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

  // Overlay rendering.

  function appendBox(root, boxData) {
    var rect = boxData.rect;
    if (rect.width < 1 || rect.height < 1) {
      return;
    }

    var color = colors[boxData.meta.kind] || "#7c3aed";
    var inset = Math.min(boxData.track * 3, Math.floor(Math.min(rect.width, rect.height) / 4));
    var frame = document.createElement("div");
    frame.setAttribute("data-goldr-template-stack", String(boxData.track));
    setStyle(frame, [
      "position:fixed",
      "left:" + (rect.left + inset) + "px",
      "top:" + (rect.top + inset) + "px",
      "width:" + Math.max(0, rect.width - inset * 2) + "px",
      "height:" + Math.max(0, rect.height - inset * 2) + "px",
      "box-sizing:border-box",
      "border:2px solid " + color,
      "pointer-events:none"
    ]);

    root.appendChild(frame);
    appendLabel(root, rect, boxData, inset);
  }

  function appendLabel(root, rect, box, inset) {
    var color = colors[box.meta.kind] || "#7c3aed";
    var expanded = expandedKey === box.key;
    var label = document.createElement("div");
    label.setAttribute("data-goldr-template-label", "1");
    setStyle(label, [
      "position:fixed",
      "left:" + (rect.left + inset - 2) + "px",
      "top:" + (rect.top + inset - 2) + "px",
      "z-index:1",
      "max-width:min(520px,calc(100% - 8px),90vw)",
      "display:flex",
      "flex-direction:column",
      "align-items:stretch",
      "gap:2px",
      "background:" + color,
      "color:white",
      "font:11px/15px system-ui,-apple-system,BlinkMacSystemFont,\"Segoe UI\",sans-serif",
      "padding:2px 4px",
      "border-radius:2px",
      "pointer-events:auto"
    ]);

    var header = document.createElement("div");
    setStyle(header, [
      "display:flex",
      "align-items:center",
      "gap:4px",
      "min-width:0"
    ]);
    appendExpandButton(header, box, expanded);

    appendBadgeText(header, box);
    var path = sourcePath(box);
    if (path) {
      appendCopyButton(header, path, box.meta.kind === "component");
    }
    label.appendChild(header);
    if (expanded) {
      appendDetails(label, box, path);
    }
    root.appendChild(label);
  }

  function appendBadgeText(root, box) {
    var parts = badgeParts(box);
    var text = document.createElement("span");
    text.setAttribute("data-goldr-template-badge-text", "1");
    setStyle(text, [
      "display:flex",
      "align-items:baseline",
      "gap:4px",
      "min-width:0",
      "overflow:hidden"
    ]);

    if (parts.kind) {
      appendStyledText(text, parts.kind + " ", "kind");
    }
    appendStyledText(text, parts.primary, "primary");
    if (parts.secondary) {
      appendStyledText(text, ": ", "separator");
      appendStyledText(text, parts.secondary, "path");
    }
    root.appendChild(text);
  }

  function appendStyledText(root, value, role) {
    var text = document.createElement("span");
    text.textContent = value;
    text.setAttribute("data-goldr-template-text", role);
    setStyle(text, styledTextStyle(role));
    root.appendChild(text);
  }

  function appendExpandButton(label, box, expanded) {
    var title = expanded ? "Hide inspector details" : "Show inspector details";
    var button = badgeButton(expanded ? "v" : ">", title, "data-goldr-template-expand");
    button.setAttribute("aria-expanded", expanded ? "true" : "false");
    button.addEventListener("click", function (event) {
      event.preventDefault();
      event.stopPropagation();
      expandedKey = expanded ? null : box.key;
      draw();
    });
    label.appendChild(button);
  }

  function appendDetails(label, box, path) {
    var details = document.createElement("div");
    details.setAttribute("data-goldr-template-details", "1");
    setStyle(details, [
      "display:grid",
      "grid-template-columns:max-content minmax(0,1fr)",
      "column-gap:6px",
      "row-gap:1px",
      "padding-top:2px",
      "border-top:1px solid rgba(255,255,255,.28)",
      "max-width:min(520px,90vw)"
    ]);

    if (box.meta.kind === "component") {
      appendDetailRow(details, "component", box.meta.label || "component");
      var context = renderContext(box);
      if (path) {
        appendPathDetailRow(details, "source context", path);
      }
      if (context) {
        appendMetaDetailRow(details, "rendered in", context);
      }
    } else {
      appendDetailRow(details, "unit", box.meta.kind || "template");
      if (box.meta.route) {
        appendPathDetailRow(details, "route pattern", box.meta.route, "chain-path");
      }
      if (box.meta.handler) {
        appendDetailRow(details, "handler", box.meta.handler);
      }
      if (path) {
        appendPathDetailRow(details, "source", path);
      }
    }
    var chain = renderChain(box);
    if (chain.length > 0) {
      appendRenderChainRow(details, "render chain", chain);
    }
    label.appendChild(details);
  }

  function appendDetailKey(details, name) {
    var key = document.createElement("span");
    key.textContent = name;
    setStyle(key, [
      "opacity:.82",
      "white-space:nowrap"
    ]);
    details.appendChild(key);
  }

  function appendDetailRow(details, name, value) {
    appendDetailKey(details, name);

    var text = document.createElement("span");
    text.textContent = value;
    setStyle(text, [
      "min-width:0",
      "overflow:hidden",
      "text-overflow:ellipsis",
      "white-space:nowrap"
    ]);
    details.appendChild(text);
  }

  function appendDetailValue(details) {
    var value = document.createElement("span");
    setStyle(value, [
      "display:flex",
      "align-items:baseline",
      "gap:4px",
      "min-width:0",
      "overflow:visible",
      "flex-wrap:wrap",
      "row-gap:2px"
    ]);
    details.appendChild(value);
    return value;
  }

  function appendPathDetailRow(details, name, path, role) {
    appendDetailKey(details, name);
    var value = appendDetailValue(details);
    appendStyledText(value, path, role || "path");
  }

  function appendMetaDetailRow(details, name, meta) {
    appendDetailKey(details, name);
    var value = appendDetailValue(details);
    appendMetaValue(value, meta);
  }

  function appendRenderChainRow(details, name, chain) {
    appendDetailKey(details, name);
    var value = document.createElement("span");
    value.setAttribute("data-goldr-template-chain", "1");
    setStyle(value, [
      "display:flex",
      "flex-direction:column",
      "gap:3px",
      "min-width:0"
    ]);
    details.appendChild(value);

    chain.forEach(function (meta, index) {
      appendRenderChainItem(value, meta, index);
    });
  }

  function appendRenderChainItem(root, meta, depth) {
    var item = document.createElement("span");
    item.setAttribute("data-goldr-template-chain-item", "1");
    setStyle(item, [
      "display:flex",
      "align-items:flex-start",
      "gap:4px",
      "min-width:0",
      "padding-left:" + Math.min(depth * 12, 72) + "px"
    ]);

    if (depth > 0) {
      var branch = document.createElement("span");
      branch.setAttribute("aria-hidden", "true");
      setStyle(branch, [
        "flex:0 0 10px",
        "height:8px",
        "border-left:1px solid rgba(255,255,255,.42)",
        "border-bottom:1px solid rgba(255,255,255,.42)",
        "margin-top:2px"
      ]);
      item.appendChild(branch);
    }

    var body = document.createElement("span");
    setStyle(body, [
      "display:flex",
      "align-items:baseline",
      "gap:4px",
      "min-width:0",
      "flex-wrap:wrap",
      "row-gap:2px"
    ]);
    appendMetaValue(body, meta, "chain");
    item.appendChild(body);
    root.appendChild(item);
  }

  function appendMetaValue(root, meta, context) {
    var pathRole = context === "chain" ? "chain-path" : "path";
    if (meta.kind === "component") {
      appendStyledText(root, "component ", "kind");
      appendStyledText(root, meta.label || "component", "component-label");
      return;
    }
    appendStyledText(root, meta.kind || "template", "kind");
    if (meta.route) {
      appendStyledText(root, " ", "separator");
      appendStyledText(root, meta.route, pathRole);
      return;
    }
    var path = sourcePathFromMeta(meta);
    if (path) {
      appendStyledText(root, " ", "separator");
      appendStyledText(root, path, pathRole);
    }
  }

  function appendCopyButton(label, path, sourceContext) {
    var title = sourceContext ? "Copy source context path to clipboard" : "Copy source path to clipboard";
    var button = badgeButton("\u29c9", title, "data-goldr-template-copy");
    button.addEventListener("click", function (event) {
      event.preventDefault();
      event.stopPropagation();
      copySourcePath(path, button, sourceContext);
    });
    label.appendChild(button);
  }

  function copySourcePath(path, button, sourceContext) {
    var title = sourceContext ? "Copy source context path to clipboard" : "Copy source path to clipboard";
    function temporaryTitle(value) {
      button.title = value;
      button.setAttribute("aria-label", value);
      window.setTimeout(function () {
        button.title = title;
        button.setAttribute("aria-label", title);
      }, 1200);
    }

    function copied() {
      temporaryTitle(sourceContext ? "Copied source context path" : "Copied source path");
    }

    function copyFailed() {
      temporaryTitle(sourceContext ? "Could not copy source context path" : "Could not copy source path");
    }

    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(path).then(copied, function () {
        if (fallbackCopySourcePath(path)) {
          copied();
          return;
        }
        copyFailed();
      });
      return;
    }

    if (fallbackCopySourcePath(path)) {
      copied();
      return;
    }
    copyFailed();
  }

  function fallbackCopySourcePath(path) {
    var input = document.createElement("textarea");
    input.value = path;
    input.setAttribute("readonly", "readonly");
    setStyle(input, [
      "position:fixed",
      "left:-9999px",
      "top:0"
    ]);
    document.body.appendChild(input);
    input.select();
    try {
      return document.execCommand("copy");
    } catch (_) {
      return false;
    } finally {
      input.remove();
    }
  }

  // Controls.

  function controlButtonStyle(active, disabled) {
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
      ];
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
    ];
  }

  function controlButton(label, title, active, disabled, extraStyle) {
    var button = document.createElement("button");
    button.type = "button";
    button.textContent = label;
    button.title = title;
    button.disabled = disabled;
    button.setAttribute("aria-label", title);
    button.setAttribute("aria-pressed", active ? "true" : "false");
    setStyle(button, controlButtonStyle(active, disabled).concat(extraStyle || []));
    return button;
  }

  function appendModeButton(root, label, value) {
    var active = mode === value && selectedIndex === null;
    var button = controlButton(label, "Show " + label.toLowerCase() + " template inspector overlays", active, false);
    button.setAttribute(modeAttribute, value);
    button.addEventListener("click", function () {
      mode = value;
      selectedIndex = null;
      expandedKey = null;
      persistMode(value);
      draw();
    });
    root.appendChild(button);
  }

  function appendNextButton(root) {
    var disabled = mode === "off";
    var active = mode !== "off" && selectedIndex !== null;
    var button = controlButton("Next", "Show the next template inspector overlay", active, disabled, ["margin-left:10px"]);
    button.setAttribute(nextAttribute, "1");
    button.addEventListener("click", function () {
      var boxes = trackedBoxes();
      if (boxes.length > 0) {
        if (selectedIndex === null) {
          selectedIndex = 0;
        } else {
          selectedIndex = (selectedIndex + 1) % boxes.length;
        }
        mode = "all";
        expandedKey = null;
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

    setStyle(root, [
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
    ]);

    root.textContent = "";
    appendModeButton(root, "All", "all");
    appendModeButton(root, "Off", "off");
    appendNextButton(root);
  }

  // Layout and redraw.

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
      expandedKey = null;
      return;
    }
    if (expandedKey && !boxesToDraw.some(function (box) { return box.key === expandedKey; })) {
      expandedKey = null;
    }

    var root = document.createElement("div");
    root.setAttribute(overlayAttribute, "1");
    setStyle(root, [
      "position:fixed",
      "inset:0",
      "pointer-events:none",
      "z-index:" + overlayZIndex
    ]);

    boxesToDraw.forEach(function (box) {
      appendBox(root, box);
    });

    if (root.childNodes.length > 0) {
      document.body.appendChild(root);
      layoutLabels(root);
    }
  }

  function scheduleDraw() {
    window.clearTimeout(scheduleDraw.timer);
    scheduleDraw.timer = window.setTimeout(function () {
      if (window.requestAnimationFrame) {
        window.requestAnimationFrame(draw);
        return;
      }
      draw();
    }, 50);
  }

  // DOM change watching.

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
