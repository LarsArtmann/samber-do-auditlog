import type { Feature } from "./types";

export const features: Feature[] = [
  {
    icon: "hook",
    title: "Drop-in Setup",
    desc: "One line: do.NewWithOpts(plugin.Opts()). Zero config, zero ceremony.",
  },
  {
    icon: "graph",
    title: "Dependency Graph",
    desc: "Infers which service resolved which — forward and reverse — without touching do's internals.",
  },
  {
    icon: "scope",
    title: "Scope Tree",
    desc: "Full hierarchy with per-scope service lists, cross-scope resolution tracked automatically.",
  },
  {
    icon: "export",
    title: "9+ Export Formats",
    desc: "JSON, NDJSON, CSV, TSV, HTML, Mermaid, PlantUML, DOT, D2, tree, and table.",
  },
  {
    icon: "health",
    title: "Health Check Audit",
    desc: "Wraps injector.HealthCheck() with per-service events and unhealthy-service detection.",
  },
  {
    icon: "toggle",
    title: "~1.7us Overhead",
    desc: "In-memory capture during operation. Toggle off for zero cost. Export only when you need it.",
  },
];
