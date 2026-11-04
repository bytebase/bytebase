import { Engine } from "@/types/proto-es/v1/common_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import catalog from "./standardRuleCatalog.json";

// standardRuleCatalog.json is recorded from omni and the backend's reviewed
// engines; backend/runner/reviewrun/standard_rule_catalog_test.go fails when
// it drifts.

export interface FindingCodeInfo {
  // Such as "WT101".
  code: string;
  // A short English title, as omni names the code.
  title: string;
}

interface CatalogRule {
  engines: string[];
  codes: FindingCodeInfo[];
}

const RULES: Readonly<Record<string, CatalogRule | undefined>> = catalog.rules;

const toEngines = (names: readonly string[]): Engine[] =>
  names.map((name) => Engine[name as keyof typeof Engine]);

// Engines the standard rule review covers. A target on any other engine is
// reviewed by the plan checks' review policy instead.
export const RULE_REVIEW_ENGINES: readonly Engine[] = toEngines(
  catalog.engines
);

// The engines that check a rule. An engine left out of a rule it lists, such
// as SQL Server for online migration, only reports that it has no such
// feature.
export const standardRuleEngines = (rule: ReviewRuleType): Engine[] =>
  toEngines(RULES[ReviewRuleType[rule]]?.engines ?? []);

// The finding codes a rule reports, in omni's order.
export const standardRuleCodes = (
  rule: ReviewRuleType
): readonly FindingCodeInfo[] => RULES[ReviewRuleType[rule]]?.codes ?? [];

const CODE_TITLES = new Map(
  Object.values(RULES).flatMap((rule) =>
    (rule?.codes ?? []).map(({ code, title }) => [code, title] as const)
  )
);

// A finding code's title, or undefined for a code this build does not know.
export const findingCodeTitle = (code: string): string | undefined =>
  CODE_TITLES.get(code);
