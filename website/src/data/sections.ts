import type { StepCard, ComparisonItem, UseCase } from "./types";

export const steps: StepCard[] = [
  {
    step: "1",
    title: "Create the Plugin",
    desc: "Pass plugin.Opts() to do.NewWithOpts. Hooks fire on every registration, invocation, and shutdown.",
  },
  {
    step: "2",
    title: "Run Your Container",
    desc: "The plugin captures events in memory with an invocation stack that infers the dependency graph.",
  },
  {
    step: "3",
    title: "Export & Visualize",
    desc: "Get a JSON snapshot, NDJSON event stream, or a self-contained HTML page. Or render Mermaid, DOT, D2, PlantUML.",
  },
];

export const comparisons: ComparisonItem[] = [
  {
    variant: "DIY",
    price: "DIY",
    accent: false,
    pros: [],
    cons: ["No built-in hooks consumer", "Manual timestamp wiring", "No dependency graph inference", "No export formats"],
  },
  {
    variant: "do-auditlog",
    price: "Free",
    accent: true,
    pros: [
      "One-line setup",
      "Auto-inferred dependency graph",
      "9+ export formats including HTML",
      "Health check audit",
      "Real-time event streaming",
    ],
    cons: [],
  },
  {
    variant: "Manual",
    price: "Manual",
    accent: false,
    pros: [],
    cons: ["Hand-roll logging per service", "No visualization", "No replay or migration", "Maintenance burden"],
  },
];

export const useCases: UseCase[] = [
  {
    title: "Debugging",
    desc: "Spot circular dependencies and slow builds instantly",
    icon: "bug",
  },
  {
    title: "Observability",
    desc: "Stream events to Prometheus, OTel, or live dashboards",
    icon: "chart",
  },
  {
    title: "CI / CD",
    desc: "Export reports as build artifacts for audit trails",
    icon: "refresh",
  },
  {
    title: "Performance",
    desc: "Profile first-build durations and invocation counts",
    icon: "bolt",
  },
];
