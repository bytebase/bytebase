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
  isStandardReviewOn,
  reviewRulesOfPolicy,
  STANDARD_RULE_TYPES,
  sameStandardRules,
  sortStandardRules,
  standardRuleText,
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

  test("read the rules of a policy, and nothing from a policy without one", () => {
    expect(reviewRulesOfPolicy(undefined)).toBeUndefined();
    // What the store caches when a project has no policy of its own.
    expect(
      reviewRulesOfPolicy(
        create(PolicySchema, { name: "projects/p/policies/review_rule" })
      )
    ).toBeUndefined();
    expect(
      reviewRulesOfPolicy(
        create(PolicySchema, {
          type: PolicyType.REVIEW_RULE,
          policy: {
            case: "reviewRulePolicy",
            value: create(ReviewRulePolicySchema, {
              rules: [ReviewRuleType.DISALLOW_RENAME, ReviewRuleType.SYNTAX],
            }),
          },
        })
      )
    ).toEqual([ReviewRuleType.SYNTAX, ReviewRuleType.DISALLOW_RENAME]);
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
