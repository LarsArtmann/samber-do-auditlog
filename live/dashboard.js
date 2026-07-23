/* live/dashboard.js — SSE client + incremental rendering engine for DI container */

(function () {
  "use strict";

  // === Utilities ===

  function esc(s) {
    if (s == null) return "";
    return String(s).replace(/[&<>"']/g, function (c) {
      return { "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[c];
    });
  }

  function humanizeDuration(ms) {
    if (ms == null || ms < 0) return "\u2014";
    if (ms < 1) return ms.toFixed(3) + "ms";
    if (ms < 1000) return ms.toFixed(1) + "ms";
    var s = ms / 1000;
    if (s < 60) return s.toFixed(1) + "s";
    var m = Math.floor(s / 60);
    var rem = s - m * 60;
    if (m < 60) return m + "m " + Math.round(rem) + "s";
    var h = Math.floor(m / 60);
    var remM = m - h * 60;
    return h + "h " + remM + "m";
  }

  // === Type metadata ===

  var meta = {};
  try {
    meta = JSON.parse(document.getElementById("type-metadata").textContent);
  } catch (e) {
    meta = { providers: {}, statuses: {}, events: {} };
  }

  var typeIcons = {};
  var typeLabels = {};
  if (meta.providers) {
    Object.keys(meta.providers).forEach(function (k) {
      typeIcons[k] = meta.providers[k].icon;
      typeLabels[k] = meta.providers[k].label;
    });
  }
  var statusIcons = {};
  if (meta.statuses) {
    Object.keys(meta.statuses).forEach(function (k) {
      statusIcons[k] = meta.statuses[k].icon;
    });
  }
  var eventLabels = {};
  var eventColors = {};
  if (meta.events) {
    Object.keys(meta.events).forEach(function (k) {
      eventLabels[k] = meta.events[k].label;
      eventColors[k] = meta.events[k].color;
    });
  }

  // === State ===

  var state = {
    report: null,
    events: [],
    services: {},
    dag: null,
    complete: false,
    maxEventSeq: 0,
  };

  // === DOM refs ===

  var els = {
    containerId: document.getElementById("container-id"),
    connStatus: document.getElementById("connection-status"),
    liveBadge: document.getElementById("live-badge"),
    legend: document.getElementById("legend"),
    stats: document.getElementById("stats"),
    waveform: document.getElementById("waveform"),
    servicesTbody: document.getElementById("services-tbody"),
    servicesEmpty: document.getElementById("services-empty"),
    serviceSearch: document.getElementById("service-search"),
    serviceResultCount: document.getElementById("service-result-count"),
    eventsTbody: document.getElementById("events-tbody"),
    eventsEmpty: document.getElementById("events-empty"),
    eventFilters: document.getElementById("event-filters"),
    graphContainer: document.getElementById("graph-container"),
    graphPlaceholder: document.getElementById("graph-placeholder"),
    timelineContainer: document.getElementById("timeline-container"),
    footerTs: document.getElementById("footer-ts"),
    footerStats: document.getElementById("footer-stats"),
  };

  // === Connection status ===

  function setConnStatus(cls, text) {
    els.connStatus.className = "conn-status " + cls;
    els.connStatus.textContent = text;
  }

  // === SSE Connection ===

  var eventSource = null;
  var reconnectDelay = 1000;
  var maxReconnectDelay = 10000;

  function connect() {
    setConnStatus("connecting", "connecting...");

    eventSource = new EventSource("/api/events");

    eventSource.addEventListener("snapshot", function (e) {
      setConnStatus("connected", "connected");
      reconnectDelay = 1000;
      handleSnapshot(JSON.parse(e.data));
    });

    eventSource.addEventListener("event", function (e) {
      handleEvent(JSON.parse(e.data));
    });

    eventSource.addEventListener("complete", function (e) {
      handleComplete(JSON.parse(e.data));
    });

    eventSource.onopen = function () {
      setConnStatus("connected", "connected");
      reconnectDelay = 1000;
    };

    eventSource.onerror = function () {
      if (state.complete) {
        setConnStatus("disconnected", "disconnected");
        if (eventSource) eventSource.close();
        return;
      }
      setConnStatus("reconnecting", "reconnecting...");
      if (eventSource) eventSource.close();
      setTimeout(function () {
        reconnectDelay = Math.min(reconnectDelay * 1.5, maxReconnectDelay);
        connect();
      }, reconnectDelay);
    };
  }

  // === Event Handlers ===

  function handleSnapshot(data) {
    state.report = data.report;
    state.events = data.events || [];
    state.dag = data.dag;
    state.complete = data.complete || false;
    state.maxEventSeq = 0;

    state.events.forEach(function (evt) {
      if (evt.sequence > state.maxEventSeq) {
        state.maxEventSeq = evt.sequence;
      }
    });

    if (state.report && state.report.services) {
      state.report.services.forEach(function (svc) {
        var key = svc.scope_id + "/" + svc.service_name;
        state.services[key] = svc;
      });
    }

    if (state.complete) {
      markComplete();
    }

    scheduleFullRender();
  }

  function handleEvent(evt) {
    if (evt.sequence <= state.maxEventSeq) return;
    state.maxEventSeq = evt.sequence;

    state.events.push(evt);

    if (evt.event_type === "registration") {
      var regKey = evt.scope_id + "/" + evt.service_name;
      if (!state.services[regKey]) {
        state.services[regKey] = {
          scope_id: evt.scope_id,
          scope_name: evt.scope_name,
          service_name: evt.service_name,
          service_type: evt.service_type || "",
          status: "registered",
          invocation_count: 0,
          first_build_duration_ms: null,
          dependencies: [],
          dependents: [],
        };
      }
    } else if (evt.event_type === "invocation" && evt.phase === "after") {
      var invKey = evt.scope_id + "/" + evt.service_name;
      if (state.services[invKey]) {
        state.services[invKey].status = evt.error ? "invocation_error" : "active";
        state.services[invKey].invocation_count =
          (state.services[invKey].invocation_count || 0) + 1;
        if (evt.duration_ms != null && !state.services[invKey].first_build_duration_ms) {
          state.services[invKey].first_build_duration_ms = evt.duration_ms;
        }
      }
    } else if (evt.event_type === "shutdown" && evt.phase === "after") {
      var shdKey = evt.scope_id + "/" + evt.service_name;
      if (state.services[shdKey]) {
        state.services[shdKey].status = evt.error ? "shutdown_error" : "shutdown";
      }
    }

    scheduleRender();
  }

  function handleComplete(data) {
    state.report = data.report;
    state.dag = data.dag;
    state.complete = true;

    if (state.report && state.report.services) {
      state.report.services.forEach(function (svc) {
        var key = svc.scope_id + "/" + svc.service_name;
        state.services[key] = svc;
      });
    }

    markComplete();
    scheduleFullRender();
  }

  function markComplete() {
    document.body.classList.add("lifecycle-complete");
    els.liveBadge.classList.add("completed");
    els.liveBadge.innerHTML = '<span class="live-pulse"></span>DONE';
    setConnStatus("connected", "complete");
  }

  // === Render scheduling (debounced via requestAnimationFrame) ===

  var renderQueued = false;

  function scheduleRender() {
    if (renderQueued) return;
    renderQueued = true;
    requestAnimationFrame(function () {
      renderQueued = false;
      renderAll();
    });
  }

  var fullRenderQueued = false;

  function scheduleFullRender() {
    if (fullRenderQueued) return;
    fullRenderQueued = true;
    requestAnimationFrame(function () {
      fullRenderQueued = false;
      renderAll();
      renderGraph();
      renderTimeline();
    });
  }

  function renderAll() {
    renderHeader();
    renderStats();
    renderWaveform();
    renderLegend();
    renderServicesTable();
    renderEventsTable();
    renderFooter();
  }

  // === Header ===

  function renderHeader() {
    if (!state.report) return;
    var r = state.report;
    els.containerId.textContent = r.container_id || "unknown";
  }

  // === Stats ===

  function renderStats() {
    if (!state.report) return;
    var r = state.report;

    var errorCount = 0;
    Object.keys(state.services).forEach(function (k) {
      var s = state.services[k];
      if (s.status === "invocation_error" || s.status === "shutdown_error") errorCount++;
    });

    var stats = [
      { label: "Services", value: r.service_count },
      { label: "Events", value: r.event_count },
      { label: "Scopes", value: r.scope_count },
      { label: "Errors", value: errorCount, cls: errorCount > 0 ? "error" : "success" },
      { label: "Build (ms)", value: humanizeDuration(r.total_build_duration_ms) },
      { label: "Shutdown (ms)", value: humanizeDuration(r.total_shutdown_duration_ms) },
    ];

    if (r.health_checked_count > 0) {
      stats.push({
        label: "Health",
        value: r.health_check_succeeded ? "Pass" : "Fail",
        cls: r.health_check_succeeded ? "success" : "error",
      });
    }

    els.stats.innerHTML = stats
      .map(function (s) {
        return (
          '<div class="stat-card' +
          (s.cls ? " " + s.cls : "") +
          '"><div class="label">' +
          s.label +
          '</div><div class="value">' +
          s.value +
          "</div></div>"
        );
      })
      .join("");
  }

  // === Waveform ===

  function renderWaveform() {
    var events = state.events;
    if (!events.length) return;

    var waveform = els.waveform;
    var placeholder = waveform.querySelector(".waveform-placeholder");
    if (placeholder) placeholder.remove();

    var ts = events.map(function (e) {
      return new Date(e.timestamp).getTime();
    });
    var minT = Math.min.apply(null, ts);
    var maxT = Math.max.apply(null, ts);
    var range = maxT - minT || 1;
    var maxDur = Math.max.apply(
      null,
      events
        .filter(function (e) {
          return e.duration_ms;
        })
        .map(function (e) {
          return e.duration_ms;
        })
        .concat([1]),
    );

    waveform.innerHTML = events
      .map(function (e) {
        var t = new Date(e.timestamp).getTime();
        var pct = ((t - minT) / range) * 100;
        var hasErr = e.error != null;
        var color = hasErr ? "var(--error)" : eventColors[e.event_type] || "var(--text-muted)";
        var dur = e.duration_ms || 0;
        var h = dur > 0 ? Math.max(4, (dur / maxDur) * 28) : 4;
        var tip =
          e.event_type +
          (e.service_name ? " " + e.service_name : "") +
          (e.phase ? " " + e.phase : "") +
          (dur > 0 ? " " + humanizeDuration(dur) : "");
        return (
          '<div class="wf-event" style="left:' +
          pct.toFixed(2) +
          "%;height:" +
          h.toFixed(0) +
          "px;background:" +
          color +
          '" title="' +
          esc(tip) +
          '"></div>'
        );
      })
      .join("");
  }

  // === Legend ===

  function renderLegend() {
    var counts = {};
    Object.keys(state.services).forEach(function (k) {
      var s = state.services[k];
      if (s.service_type) counts[s.service_type] = (counts[s.service_type] || 0) + 1;
    });

    var order = ["lazy", "eager", "transient", "alias"];
    els.legend.innerHTML = order
      .map(function (k) {
        var count = counts[k] || 0;
        return count > 0
          ? '<div class="legend-item"><span class="icon">' +
              (typeIcons[k] || "") +
              "</span>" +
              esc(typeLabels[k] || k) +
              ' <span style="opacity:0.5">(' +
              count +
              ")</span></div>"
          : "";
      })
      .join("");
  }

  // === Services Table ===

  function renderServicesTable() {
    var keys = Object.keys(state.services);
    if (!keys.length) {
      els.servicesTbody.innerHTML = "";
      els.servicesEmpty.style.display = "";
      return;
    }

    els.servicesEmpty.style.display = "none";

    var searchTerm = els.serviceSearch ? els.serviceSearch.value.toLowerCase() : "";
    var filtered = keys.filter(function (k) {
      if (!searchTerm) return true;
      var s = state.services[k];
      return (
        s.service_name.toLowerCase().indexOf(searchTerm) >= 0 ||
        s.scope_name.toLowerCase().indexOf(searchTerm) >= 0
      );
    });

    if (els.serviceResultCount) {
      els.serviceResultCount.textContent = filtered.length + " of " + keys.length;
    }

    filtered.sort(function (a, b) {
      var sa = state.services[a],
        sb = state.services[b];
      if (sa.scope_name !== sb.scope_name) return sa.scope_name < sb.scope_name ? -1 : 1;
      return sa.service_name < sb.service_name ? -1 : 1;
    });

    els.servicesTbody.innerHTML = filtered
      .map(function (k) {
        var s = state.services[k];
        var icon = typeIcons[s.service_type] || "";
        var statusIcon = statusIcons[s.status] || "";
        var depNames = (s.dependencies || [])
          .map(function (d) {
            return esc(typeof d === "string" ? d : d.service_name);
          })
          .join(", ");
        var dependentNames = (s.dependents || [])
          .map(function (d) {
            return esc(typeof d === "string" ? d : d.service_name);
          })
          .join(", ");
        var buildMs =
          s.first_build_duration_ms != null
            ? humanizeDuration(s.first_build_duration_ms)
            : "\u2014";
        var invocErr = s.invocation_error ? esc(s.invocation_error) : "";
        var shutdownErr = s.shutdown_error ? esc(s.shutdown_error) : "";
        var error = invocErr || shutdownErr || "";

        return (
          "<tr>" +
          "<td>" +
          icon +
          " " +
          esc(s.service_name) +
          "</td>" +
          "<td>" +
          esc(s.scope_name) +
          "</td>" +
          "<td>" +
          esc(s.service_type || "") +
          "</td>" +
          "<td>" +
          statusIcon +
          " " +
          esc(s.status || "") +
          "</td>" +
          "<td>" +
          (s.invocation_count || 0) +
          "</td>" +
          "<td>" +
          buildMs +
          "</td>" +
          "<td>" +
          depNames +
          "</td>" +
          "<td title='" +
          esc(error) +
          "'>" +
          (error ? "\u26A0 " + error.substring(0, 40) : "\u2014") +
          "</td>" +
          "</tr>"
        );
      })
      .join("");
  }

  // === Events Table ===

  function renderEventsTable() {
    if (!state.events.length) {
      els.eventsTbody.innerHTML = "";
      els.eventsEmpty.style.display = "";
      return;
    }

    els.eventsEmpty.style.display = "none";

    var activeFilter = els.eventFilters ? els.eventFilters.querySelector(".chip.active") : null;
    var filterType = activeFilter ? activeFilter.getAttribute("data-type") : null;

    var filtered = state.events;
    if (filterType) {
      filtered = filtered.filter(function (e) {
        return e.event_type === filterType;
      });
    }

    els.eventsTbody.innerHTML = filtered
      .map(function (e, idx) {
        var label = eventLabels[e.event_type] || e.event_type;
        var color = eventColors[e.event_type] || "var(--text-muted)";
        var phase = e.phase === "before" ? "\u25B2" : "\u25BE";
        var dur = e.duration_ms != null ? humanizeDuration(e.duration_ms) : "\u2014";
        var err = e.error ? esc(e.error) : "";

        return (
          "<tr>" +
          "<td>" +
          (idx + 1) +
          "</td>" +
          "<td>" +
          new Date(e.timestamp).toLocaleTimeString() +
          "</td>" +
          "<td><span class='event-badge' style='background:" +
          color +
          "'>" +
          esc(label) +
          "</span></td>" +
          "<td>" +
          phase +
          " " +
          esc(e.phase) +
          "</td>" +
          "<td>" +
          esc(e.service_name) +
          "</td>" +
          "<td>" +
          dur +
          "</td>" +
          "<td title='" +
          esc(err) +
          "'>" +
          (err ? "\u26A0 " + err.substring(0, 30) : "") +
          "</td>" +
          "</tr>"
        );
      })
      .join("");
  }

  // === Graph (placeholder — uses daghtml.Script() from Go) ===

  function renderGraph() {
    if (!state.dag) return;
    if (els.graphPlaceholder) els.graphPlaceholder.style.display = "none";
  }

  // === Timeline (placeholder) ===

  function renderTimeline() {
    if (!state.events.length) return;
    if (els.timelineContainer) {
      var placeholder = els.timelineContainer.querySelector(".graph-placeholder");
      if (placeholder) placeholder.remove();
    }
  }

  // === Footer ===

  function renderFooter() {
    if (els.footerTs) {
      els.footerTs.textContent = new Date().toLocaleString();
    }
    if (els.footerStats && state.report) {
      els.footerStats.textContent =
        "Schema v" +
        (state.report.version || "?") +
        " | " +
        state.events.length +
        " events | " +
        Object.keys(state.services).length +
        " services";
    }
  }

  // === Service search ===

  if (els.serviceSearch) {
    els.serviceSearch.addEventListener("input", function () {
      scheduleRender();
    });
  }

  // === Event filter chips ===

  function setupEventFilters() {
    if (!els.eventFilters) return;
    var types = ["registration", "invocation", "shutdown", "health_check"];
    els.eventFilters.innerHTML = types
      .map(function (t) {
        var label = eventLabels[t] || t;
        var color = eventColors[t] || "var(--text-muted)";
        return (
          "<button class='chip' data-type='" +
          t +
          "' style='border-color:" +
          color +
          "'>" +
          esc(label) +
          "</button>"
        );
      })
      .join("");

    els.eventFilters.addEventListener("click", function (e) {
      var chip = e.target.closest(".chip");
      if (!chip) return;

      var isActive = chip.classList.contains("active");
      els.eventFilters.querySelectorAll(".chip").forEach(function (c) {
        c.classList.remove("active");
      });
      if (!isActive) chip.classList.add("active");

      renderEventsTable();
    });
  }

  // === Keyboard navigation ===

  document.addEventListener("keydown", function (e) {
    if (e.target.tagName === "INPUT") return;
    var tabs = document.querySelectorAll(".tab[data-tab]");
    var num = parseInt(e.key, 10);
    if (num >= 1 && num <= tabs.length) {
      tabs[num - 1].click();
    }
  });

  // === Tab switching ===

  document.querySelectorAll(".tab[data-tab]").forEach(function (tab) {
    tab.addEventListener("click", function () {
      document.querySelectorAll(".tab").forEach(function (t) {
        t.classList.remove("active");
        t.setAttribute("aria-selected", "false");
      });
      document.querySelectorAll(".tab-content").forEach(function (c) {
        c.classList.remove("active");
      });

      tab.classList.add("active");
      tab.setAttribute("aria-selected", "true");
      var panelId = "tab-" + tab.getAttribute("data-tab");
      var panel = document.getElementById(panelId);
      if (panel) panel.classList.add("active");
    });
  });

  // === Init ===

  setupEventFilters();
  connect();
})();
