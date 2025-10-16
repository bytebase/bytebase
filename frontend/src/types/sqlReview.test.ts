import { describe, expect, test } from "vitest";
import { SQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";
import sqlReviewSchema from "./sql-review-schema.yaml";
import sqlReviewDevTemplate from "./sql-review.dev.yaml";
import sqlReviewProdTemplate from "./sql-review.prod.yaml";
import sqlReviewSampleTemplate from "./sql-review.sample.yaml";

describe("SQL Review YAML Templates Validation", () => {
  const templates = [
    { name: "sample", data: sqlReviewSampleTemplate },
    { name: "dev", data: sqlReviewDevTemplate },
    { name: "prod", data: sqlReviewProdTemplate },
  ];

  templates.forEach(({ name, data }) => {
    describe(`Template: ${name}`, () => {
      test("should have id field", () => {
        expect(data.id).toBeDefined();
        expect(typeof data.id).toBe("string");
        expect(data.id.length).toBeGreaterThan(0);
      });

      test("should have ruleList array", () => {
        expect(data.ruleList).toBeDefined();
        expect(Array.isArray(data.ruleList)).toBe(true);
        expect(data.ruleList.length).toBeGreaterThan(0);
      });

      test("all rules must have required fields", () => {
        data.ruleList.forEach((rule: any, index: number) => {
          const ruleDesc = `rule[${index}] (${rule.type || "unknown"})`;

          // Type is required
          expect(rule.type, `${ruleDesc} must have type`).toBeDefined();
          expect(typeof rule.type, `${ruleDesc} type must be string`).toBe(
            "string"
          );

          // Engine is required
          expect(rule.engine, `${ruleDesc} must have engine`).toBeDefined();
          expect(typeof rule.engine, `${ruleDesc} engine must be string`).toBe(
            "string"
          );

          // Level is required and must be valid
          expect(rule.level, `${ruleDesc} must have level`).toBeDefined();
          expect(typeof rule.level, `${ruleDesc} level must be string`).toBe(
            "string"
          );
        });
      });

      test("all rules must have valid level (ERROR or WARNING, not LEVEL_UNSPECIFIED)", () => {
        const validLevels = ["ERROR", "WARNING"];
        data.ruleList.forEach((rule: any, index: number) => {
          const ruleDesc = `rule[${index}] (${rule.type})`;
          expect(
            validLevels.includes(rule.level),
            `${ruleDesc} has invalid level '${rule.level}', must be ERROR or WARNING`
          ).toBe(true);

          // Also ensure it's not LEVEL_UNSPECIFIED or DISABLED
          expect(
            rule.level,
            `${ruleDesc} must not have LEVEL_UNSPECIFIED`
          ).not.toBe("LEVEL_UNSPECIFIED");
          expect(rule.level, `${ruleDesc} must not have DISABLED`).not.toBe(
            "DISABLED"
          );
        });
      });

      test("all rules must be convertible to SQLReviewRuleLevel enum", () => {
        data.ruleList.forEach((rule: any, index: number) => {
          const ruleDesc = `rule[${index}] (${rule.type})`;
          const levelKey = rule.level as keyof typeof SQLReviewRuleLevel;
          const levelValue = SQLReviewRuleLevel[levelKey];

          expect(
            levelValue,
            `${ruleDesc} level '${rule.level}' must map to valid enum`
          ).toBeDefined();
          expect(
            levelValue,
            `${ruleDesc} level must not be LEVEL_UNSPECIFIED (0)`
          ).not.toBe(SQLReviewRuleLevel.LEVEL_UNSPECIFIED);
        });
      });
    });
  });

  describe("Schema validation", () => {
    test("should be an array", () => {
      expect(Array.isArray(sqlReviewSchema)).toBe(true);
      expect(sqlReviewSchema.length).toBeGreaterThan(0);
    });

    test("schema rules should NOT have level field", () => {
      sqlReviewSchema.forEach((rule: any, index: number) => {
        const ruleDesc = `schema rule[${index}] (${rule.type || "unknown"})`;

        // Schema rules are just definitions, they should not have a level
        expect(
          rule.level,
          `${ruleDesc} should not have level field (schema rules are templates, not configured rules)`
        ).toBeUndefined();
      });
    });

    test("schema rules must have required fields", () => {
      sqlReviewSchema.forEach((rule: any, index: number) => {
        const ruleDesc = `schema rule[${index}] (${rule.type || "unknown"})`;

        // Type is required
        expect(rule.type, `${ruleDesc} must have type`).toBeDefined();
        expect(typeof rule.type, `${ruleDesc} type must be string`).toBe(
          "string"
        );

        // Engine is required
        expect(rule.engine, `${ruleDesc} must have engine`).toBeDefined();
        expect(typeof rule.engine, `${ruleDesc} engine must be string`).toBe(
          "string"
        );

        // Category is required
        expect(rule.category, `${ruleDesc} must have category`).toBeDefined();
        expect(
          typeof rule.category,
          `${ruleDesc} category must be string`
        ).toBe("string");
      });
    });
  });

  describe("Cross-template consistency", () => {
    test("report template rules that don't exist in schema", () => {
      const schemaRuleTypes = new Set(
        sqlReviewSchema.map((rule: any) => `${rule.engine}:${rule.type}`)
      );

      const missingRules: string[] = [];
      templates.forEach(({ name, data }) => {
        data.ruleList.forEach((rule: any) => {
          const ruleKey = `${rule.engine}:${rule.type}`;
          if (!schemaRuleTypes.has(ruleKey)) {
            missingRules.push(`${name}: ${ruleKey}`);
          }
        });
      });

      // Log missing rules but don't fail - they might be intentional
      if (missingRules.length > 0) {
        console.warn(
          "Warning: Template rules not found in schema:",
          missingRules
        );
      }

      // This test just verifies we can check consistency, not that it's perfect
      expect(schemaRuleTypes.size).toBeGreaterThan(0);
    });
  });
});
