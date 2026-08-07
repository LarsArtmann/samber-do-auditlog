/* live/dashboard.js — keyboard nav + export helpers for the datastar-powered dashboard.
 *
 * The datastar.js runtime handles all SSE transport, DOM morphing, and signal
 * reactivity. This script provides only:
 *   - handleKeydown: tab switching (1-5, arrows), search focus (/), help dialog (?)
 *   - exportReport: download triggers for JSON/NDJSON/HTML exports
 *   - Scope tree expand/collapse (click + keyboard)
 *   - Footer timestamp auto-update
 */

(function () {
  "use strict";

  var basePath = window.__LIVE_PREFIX || "";

  // === Export ===

  function exportReport(format) {
    var path = basePath + "/api/export/" + format;
    if (format === "json") path = basePath + "/api/report";
    window.location.href = path;
  }

  // === Keyboard navigation ===

  var tabNames = ["services", "scopes", "graph", "timeline", "events"];

  function isTypingElement(el) {
    if (!el || !el.tagName) return false;
    var tag = el.tagName;
    return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT";
  }

  function clickTab(name) {
    var btn = document.getElementById("tab-btn-" + name);
    if (btn) btn.click();
  }

  function getActiveTabIndex() {
    var tabs = document.querySelectorAll(".tab[data-tab]");
    for (var i = 0; i < tabs.length; i++) {
      if (tabs[i].classList.contains("active")) return i;
    }
    return 0;
  }

  function switchTabRelative(delta) {
    var tabs = document.querySelectorAll(".tab[data-tab]");
    var cur = getActiveTabIndex();
    var next = (cur + delta + tabs.length) % tabs.length;
    if (tabs[next]) tabs[next].click();
  }

  var kbdHelpPrevFocus = null;

  function closeKbdHelp() {
    var help = document.getElementById("kbd-help");
    if (!help) return;
    help.remove();
    if (kbdHelpPrevFocus) {
      kbdHelpPrevFocus.focus();
      kbdHelpPrevFocus = null;
    }
  }

  function showShortcutsHelp() {
    var existing = document.getElementById("kbd-help");
    if (existing) {
      closeKbdHelp();
      return;
    }
    kbdHelpPrevFocus = document.activeElement;
    var div = document.createElement("div");
    div.id = "kbd-help";
    div.className = "kbd-help";
    div.setAttribute("role", "dialog");
    div.setAttribute("aria-modal", "true");
    div.setAttribute("aria-label", "Keyboard shortcuts");
    div.innerHTML =
      '<div class="kbd-help-content">' +
      "<h2>Keyboard shortcuts</h2>" +
      "<ul>" +
      "<li><span>Switch to tab 1\u20135</span><kbd>1</kbd>\u2013<kbd>5</kbd></li>" +
      "<li><span>Next / previous tab</span><kbd>\u2190</kbd> <kbd>\u2192</kbd></li>" +
      "<li><span>First / last tab</span><kbd>Home</kbd> <kbd>End</kbd></li>" +
      "<li><span>Focus service search</span><kbd>/</kbd></li>" +
      "<li><span>Show this help</span><kbd>?</kbd></li>" +
      "<li><span>Close help</span><kbd>Esc</kbd></li>" +
      "</ul>" +
      '<button class="chip" id="kbd-help-close">Close</button>' +
      "</div>";
    document.body.appendChild(div);
    var closeBtn = document.getElementById("kbd-help-close");
    closeBtn.addEventListener("click", closeKbdHelp);
    closeBtn.focus();
    div.addEventListener("keydown", function (e) {
      if (e.key !== "Tab") return;
      var f = div.querySelectorAll(
        'button,[href],input,select,textarea,[tabindex]:not([tabindex="-1"])',
      );
      if (f.length === 0) return;
      if (e.shiftKey && document.activeElement === f[0]) {
        e.preventDefault();
        f[f.length - 1].focus();
      } else if (!e.shiftKey && document.activeElement === f[f.length - 1]) {
        e.preventDefault();
        f[0].focus();
      }
    });
  }

  function handleKeydown(evt) {
    var target = evt.target;
    var onTab = target.classList && target.classList.contains("tab");

    if (onTab) {
      if (evt.key === "ArrowRight") {
        evt.preventDefault();
        switchTabRelative(1);
        return;
      }
      if (evt.key === "ArrowLeft") {
        evt.preventDefault();
        switchTabRelative(-1);
        return;
      }
      if (evt.key === "Home") {
        evt.preventDefault();
        clickTab(tabNames[0]);
        return;
      }
      if (evt.key === "End") {
        evt.preventDefault();
        clickTab(tabNames[tabNames.length - 1]);
        return;
      }
    }

    if (evt.key === "Escape") {
      if (document.getElementById("kbd-help")) {
        evt.preventDefault();
        closeKbdHelp();
        return;
      }
    }

    if (isTypingElement(target)) return;

    if (evt.key === "?") {
      evt.preventDefault();
      showShortcutsHelp();
      return;
    }

    if (evt.key === "/") {
      evt.preventDefault();
      var search = document.getElementById("service-search");
      if (search) search.focus();
      return;
    }

    var tabIdx = parseInt(evt.key, 10) - 1;
    if (tabIdx >= 0 && tabIdx < tabNames.length) {
      evt.preventDefault();
      clickTab(tabNames[tabIdx]);
    }
  }

  // === Scope tree toggle (event delegation) ===

  function toggleScopeNode(header) {
    var node = header.parentElement;
    if (!node) return;
    var body = node.querySelector(":scope > .scope-body");
    if (!body) return;
    var collapsed = body.style.display === "none";
    body.style.display = collapsed ? "" : "none";
    header.setAttribute("aria-expanded", collapsed ? "true" : "false");
    var icon = header.querySelector(".scope-icon");
    if (icon) icon.textContent = collapsed ? "\u25BC" : "\u25B6";
  }

  document.addEventListener("click", function (e) {
    var header = e.target.closest(".scope-label");
    if (!header) return;
    toggleScopeNode(header);
  });

  document.addEventListener("keydown", function (e) {
    var header = e.target.closest(".scope-label");
    if (!header) return;
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      toggleScopeNode(header);
    }
  });

  // === Init tab attributes for roving tabindex ===

  function initTabAttributes() {
    var tabs = document.querySelectorAll(".tab[data-tab]");
    tabs.forEach(function (tab, i) {
      tab.setAttribute("tabindex", i === 0 ? "0" : "-1");
    });
  }

  // === Footer timestamp ===

  function updateFooterTs() {
    var el = document.getElementById("footer-ts");
    if (el) el.textContent = new Date().toLocaleString();
  }

  // === Init ===

  initTabAttributes();
  updateFooterTs();
  setInterval(updateFooterTs, 1000);

  // Expose for data-on:* attributes
  window.handleKeydown = handleKeydown;
  window.exportReport = exportReport;
})();
