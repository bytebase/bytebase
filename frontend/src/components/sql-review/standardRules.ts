import type { TFunction } from "i18next";
import type { Policy } from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";

// The standard rules in display order.
export const STANDARD_RULE_TYPES: readonly ReviewRuleType[] = [
  ReviewRuleType.SYNTAX,
  ReviewRuleType.WALK_THROUGH,
  ReviewRuleType.ONLINE_MIGRATION,
  ReviewRuleType.PRIOR_BACKUP,
  ReviewRuleType.REQUIRE_IS_NULL,
  ReviewRuleType.REQUIRE_WHERE,
  ReviewRuleType.DISALLOW_DROP_OBJECT,
  ReviewRuleType.DISALLOW_TRUNCATE,
  ReviewRuleType.DISALLOW_DROP_CONSTRAINT,
  ReviewRuleType.DISALLOW_RENAME,
  ReviewRuleType.REQUIRE_PRIMARY_KEY,
];

// Puts rules in display order, which makes two lists comparable.
export const sortStandardRules = (
  rules: readonly ReviewRuleType[]
): ReviewRuleType[] =>
  STANDARD_RULE_TYPES.filter((rule) => rules.includes(rule));

export const sameStandardRules = (
  a: readonly ReviewRuleType[],
  b: readonly ReviewRuleType[]
): boolean => {
  const left = sortStandardRules(a);
  const right = sortStandardRules(b);
  return (
    left.length === right.length &&
    left.every((rule, index) => rule === right[index])
  );
};

// SYNTAX gates the rest: without it no standard rule runs.
export const isStandardReviewOn = (rules: readonly ReviewRuleType[]): boolean =>
  rules.includes(ReviewRuleType.SYNTAX);

// The rules a resource's own policy row switches on, whatever its enforce
// flag; undefined when the resource has no row. A list without SYNTAX runs
// nothing, so it reads as empty, as the backend applies it.
export const storedReviewRules = (
  policy: Policy | undefined
): ReviewRuleType[] | undefined => {
  if (policy?.policy.case !== "reviewRulePolicy") return undefined;
  const { rules } = policy.policy.value;
  return isStandardReviewOn(rules) ? sortStandardRules(rules) : [];
};

// The rules a cached review rule policy switches on, or undefined when the
// resource has none of its own. A policy that is not enforced counts as
// absent, as the backend reads it.
export const reviewRulesOfPolicy = (
  policy: Policy | undefined
): ReviewRuleType[] | undefined =>
  policy?.enforce ? storedReviewRules(policy) : undefined;

// The rules the backend applies for the workspace: its enforced policy, else
// every rule. Undefined when the policy could not be read.
export const effectiveWorkspaceRules = (
  policy: Policy | undefined
): ReviewRuleType[] | undefined =>
  policy === undefined
    ? undefined
    : (reviewRulesOfPolicy(policy) ?? [...STANDARD_RULE_TYPES]);

export const standardRuleText = (
  t: TFunction,
  rule: ReviewRuleType
): { title: string; description: string } => {
  switch (rule) {
    case ReviewRuleType.SYNTAX:
      return {
        title: t("sql-review.standard-rules.rule.syntax.title"),
        description: t("sql-review.standard-rules.rule.syntax.description"),
      };
    case ReviewRuleType.WALK_THROUGH:
      return {
        title: t("sql-review.standard-rules.rule.walk-through.title"),
        description: t(
          "sql-review.standard-rules.rule.walk-through.description"
        ),
      };
    case ReviewRuleType.ONLINE_MIGRATION:
      return {
        title: t("sql-review.standard-rules.rule.online-migration.title"),
        description: t(
          "sql-review.standard-rules.rule.online-migration.description"
        ),
      };
    case ReviewRuleType.PRIOR_BACKUP:
      return {
        title: t("sql-review.standard-rules.rule.prior-backup.title"),
        description: t(
          "sql-review.standard-rules.rule.prior-backup.description"
        ),
      };
    case ReviewRuleType.REQUIRE_IS_NULL:
      return {
        title: t("sql-review.standard-rules.rule.require-is-null.title"),
        description: t(
          "sql-review.standard-rules.rule.require-is-null.description"
        ),
      };
    case ReviewRuleType.REQUIRE_WHERE:
      return {
        title: t("sql-review.standard-rules.rule.require-where.title"),
        description: t(
          "sql-review.standard-rules.rule.require-where.description"
        ),
      };
    case ReviewRuleType.DISALLOW_DROP_OBJECT:
      return {
        title: t("sql-review.standard-rules.rule.disallow-drop-object.title"),
        description: t(
          "sql-review.standard-rules.rule.disallow-drop-object.description"
        ),
      };
    case ReviewRuleType.DISALLOW_TRUNCATE:
      return {
        title: t("sql-review.standard-rules.rule.disallow-truncate.title"),
        description: t(
          "sql-review.standard-rules.rule.disallow-truncate.description"
        ),
      };
    case ReviewRuleType.DISALLOW_DROP_CONSTRAINT:
      return {
        title: t(
          "sql-review.standard-rules.rule.disallow-drop-constraint.title"
        ),
        description: t(
          "sql-review.standard-rules.rule.disallow-drop-constraint.description"
        ),
      };
    case ReviewRuleType.DISALLOW_RENAME:
      return {
        title: t("sql-review.standard-rules.rule.disallow-rename.title"),
        description: t(
          "sql-review.standard-rules.rule.disallow-rename.description"
        ),
      };
    case ReviewRuleType.REQUIRE_PRIMARY_KEY:
      return {
        title: t("sql-review.standard-rules.rule.require-primary-key.title"),
        description: t(
          "sql-review.standard-rules.rule.require-primary-key.description"
        ),
      };
    default:
      return { title: ReviewRuleType[rule] ?? String(rule), description: "" };
  }
};
