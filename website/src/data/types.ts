export const featureIconKeys = ["hook", "graph", "scope", "export", "health", "toggle"] as const;
export type FeatureIcon = (typeof featureIconKeys)[number];

export interface Feature {
  icon: FeatureIcon;
  title: string;
  desc: string;
}

export type ComparisonVariant = "DIY" | "do-auditlog" | "Manual";

export interface ComparisonItem {
  variant: ComparisonVariant;
  price: string;
  pros: string[];
  cons: string[];
  accent: boolean;
}

export interface StepCard {
  step: string;
  title: string;
  desc: string;
}

export const useCaseIconKeys = ["chart", "bug", "refresh", "bolt"] as const;
export type UseCaseIcon = (typeof useCaseIconKeys)[number];

export interface UseCase {
  title: string;
  desc: string;
  icon: UseCaseIcon;
}

export const uiIconKeys = [
  "arrow-external",
  "arrow-right",
  "github",
  "menu",
  "close",
  "sun",
  "moon",
  "star",
] as const;
export type UIIcon = (typeof uiIconKeys)[number];

export type IconName = FeatureIcon | UseCaseIcon | UIIcon;
