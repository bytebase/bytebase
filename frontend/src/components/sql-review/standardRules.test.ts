import { create } from "@bufbuild/protobuf";
import type { TFunction } from "i18next";
import { describe, expect, test } from "vitest";
import {
  PolicySchema,
  PolicyType,
  ReviewRulePolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { ReviewRuleType } from "@/types/proto-es/v1/review_rule_pb";
import {
  effectiveWorkspaceRules,
  isStandardReviewOn,
  reviewRulesOfPolicy,
  STANDARD_RULE_TYPES,
  sameStandardRules,
  sortStandardRules,
  standardRuleText,
  storedReviewRules,
} from "./standardRules";

const t = ((key: string) => key) as unknown as TFunction;

const everyRuleType = Object.values(ReviewRuleType).filter(
  (value): value is ReviewRuleType =>
    typeof value === "number" &&
    value !== ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED
);

describe("standard rules", () => {
  test("list every review rule type once", () => {
    expect([...STANDARD_RULE_TYPES].sort((a, b) => a - b)).toEqual(
      everyRuleType.sort((a, b) => a - b)
    );
  });

  test("give every rule its own title and description", () => {
    const keys = STANDARD_RULE_TYPES.flatMap((rule) => {
      const { title, description } = standardRuleText(t, rule);
      return [title, description];
    });
    for (const key of keys) {
      expect(key).toMatch(/^sql-review\.standard-rules\.rule\./);
    }
    expect(new Set(keys).size).toBe(keys.length);
  });

  test("compare lists regardless of order", () => {
    expect(
      sortStandardRules([
        ReviewRuleType.REQUIRE_WHERE,
        ReviewRuleType.SYNTAX,
        ReviewRuleType.REVIEW_RULE_TYPE_UNSPECIFIED,
      ])
    ).toEqual([ReviewRuleType.SYNTAX, ReviewRuleType.REQUIRE_WHERE]);
    expect(
      sameStandardRules(
        [ReviewRuleType.REQUIRE_WHERE, ReviewRuleType.SYNTAX],
        [ReviewRuleType.SYNTAX, ReviewRuleType.REQUIRE_WHERE]
      )
    ).toBe(true);
    expect(
      sameStandardRules([ReviewRuleType.SYNTAX], [ReviewRuleType.WALK_THROUGH])
    ).toBe(false);
  });

  const policy = (enforce: boolean, rules: ReviewRuleType[]) =>
    create(PolicySchema, {
      type: PolicyType.REVIEW_RULE,
      enforce,
      policy: {
        case: "reviewRulePolicy",
        value: create(ReviewRulePolicySchema, { rules }),
      },
    });
  // What the store caches when a project has no policy of its own.
  const noPolicy = create(PolicySchema, {
    name: "projects/p/policies/review_rule",
  });
  const switchedOff = policy(false, [ReviewRuleType.SYNTAX]);
  const customized = policy(true, [
    ReviewRuleType.DISALLOW_RENAME,
    ReviewRuleType.SYNTAX,
  ]);

  test("read the rules of a policy, and nothing from a policy without one", () => {
    expect(reviewRulesOfPolicy(undefined)).toBeUndefined();
    // Switched off through the API: the backend reads it as absent.
    expect(reviewRulesOfPolicy(switchedOff)).toBeUndefined();
    expect(reviewRulesOfPolicy(noPolicy)).toBeUndefined();
    expect(reviewRulesOfPolicy(customized)).toEqual([
      ReviewRuleType.SYNTAX,
      ReviewRuleType.DISALLOW_RENAME,
    ]);
  });

  test("read the stored rules of a row whether or not it is enforced", () => {
    expect(storedReviewRules(undefined)).toBeUndefined();
    expect(storedReviewRules(noPolicy)).toBeUndefined();
    expect(storedReviewRules(switchedOff)).toEqual([ReviewRuleType.SYNTAX]);
    expect(storedReviewRules(customized)).toEqual([
      ReviewRuleType.SYNTAX,
      ReviewRuleType.DISALLOW_RENAME,
    ]);
  });

  test("apply every rule for a workspace without an enforced policy", () => {
    expect(effectiveWorkspaceRules(undefined)).toBeUndefined();
    expect(effectiveWorkspaceRules(switchedOff)).toEqual(STANDARD_RULE_TYPES);
    expect(effectiveWorkspaceRules(customized)).toEqual([
      ReviewRuleType.SYNTAX,
      ReviewRuleType.DISALLOW_RENAME,
    ]);
  });

  test("need SYNTAX for standard review to run", () => {
    expect(isStandardReviewOn(STANDARD_RULE_TYPES)).toBe(true);
    expect(
      isStandardReviewOn(
        STANDARD_RULE_TYPES.filter((rule) => rule !== ReviewRuleType.SYNTAX)
      )
    ).toBe(false);
    expect(isStandardReviewOn([])).toBe(false);
  });
});
