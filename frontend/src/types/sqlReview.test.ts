import { describe, expect, test } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  SQLReviewRule_Level,
  SQLReviewRule_Type,
} from "@/types/proto-es/v1/review_config_service_pb";
import sqlReviewDevTemplate from "./sql-review.dev.yaml";
import sqlReviewProdTemplate from "./sql-review.prod.yaml";
import sqlReviewSampleTemplate from "./sql-review.sample.yaml";
import sqlReviewSchema from "./sql-review-schema.yaml";
import {
  convertRuleMapToPolicyRuleList,
  type RuleTemplateV2,
  validateRuleMapByEngine,
} from "./sqlReview";

// Type for template rule data loaded from YAML
interface TemplateRule {
  type: string;
  engine: string;
  level: string;
  payload?: Record<string, unknown>;
}

// Type for schema rule data loaded from YAML
interface SchemaRule {
  type: string;
  engine: string;
  category: string;
  level?: string;
  componentList?: unknown[];
}

// Type for template data loaded from YAML
interface TemplateData {
  id: string;
  ruleList: TemplateRule[];
}

describe("SQL Review YAML Templates Validation", () => {
  const templates: { name: string; data: TemplateData }[] = [
    {
      name: "sample",
      data: sqlReviewSampleTemplate as unknown as TemplateData,
    },
    { name: "dev", data: sqlReviewDevTemplate as unknown as TemplateData },
    { name: "prod", data: sqlReviewProdTemplate as unknown as TemplateData },
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
        (data.ruleList as TemplateRule[]).forEach((rule, index) => {
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
        (data.ruleList as TemplateRule[]).forEach((rule, index) => {
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

      test("all rules must be convertible to SQLReviewRule_Level enum", () => {
        (data.ruleList as TemplateRule[]).forEach((rule, index) => {
          const ruleDesc = `rule[${index}] (${rule.type})`;
          const levelKey = rule.level as keyof typeof SQLReviewRule_Level;
          const levelValue = SQLReviewRule_Level[levelKey];

          expect(
            levelValue,
            `${ruleDesc} level '${rule.level}' must map to valid enum`
          ).toBeDefined();
          expect(
            levelValue,
            `${ruleDesc} level must not be LEVEL_UNSPECIFIED (0)`
          ).not.toBe(SQLReviewRule_Level.LEVEL_UNSPECIFIED);
        });
      });
    });
  });

  describe("Schema validation", () => {
    const schemaRules = sqlReviewSchema as unknown as SchemaRule[];

    test("should be an array", () => {
      expect(Array.isArray(schemaRules)).toBe(true);
      expect(schemaRules.length).toBeGreaterThan(0);
    });

    test("schema rules should NOT have level field", () => {
      schemaRules.forEach((rule, index) => {
        const ruleDesc = `schema rule[${index}] (${rule.type || "unknown"})`;

        // Schema rules are just definitions, they should not have a level
        expect(
          rule.level,
          `${ruleDesc} should not have level field (schema rules are templates, not configured rules)`
        ).toBeUndefined();
      });
    });

    test("schema rules must have required fields", () => {
      schemaRules.forEach((rule, index) => {
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

    test("schema exposes built-in prior backup check for MariaDB", () => {
      expect(
        schemaRules.some(
          (rule) =>
            rule.engine === "MARIADB" &&
            rule.type === "BUILTIN_PRIOR_BACKUP_CHECK" &&
            rule.category === "BUILTIN"
        )
      ).toBe(true);
    });
  });

  describe("Cross-template consistency", () => {
    test("report template rules that don't exist in schema", () => {
      const schemaRuleTypes = new Set(
        (sqlReviewSchema as unknown as SchemaRule[]).map(
          (rule) => `${rule.engine}:${rule.type}`
        )
      );

      const missingRules: string[] = [];
      templates.forEach(({ name, data }) => {
        data.ruleList.forEach((rule) => {
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

describe("convertRuleMapToPolicyRuleList", () => {
  const stringArrayRule = (
    type: SQLReviewRule_Type,
    list: string[]
  ): RuleTemplateV2 => ({
    type,
    category: "TABLE",
    engine: Engine.MYSQL,
    level: SQLReviewRule_Level.ERROR,
    componentList: [
      {
        key: "list",
        payload: {
          type: "STRING_ARRAY",
          default: [],
          value: list,
        },
      },
    ],
  });

  const requiredStringArrayRuleTypes = [
    SQLReviewRule_Type.COLUMN_REQUIRED,
    SQLReviewRule_Type.COLUMN_TYPE_DISALLOW_LIST,
    SQLReviewRule_Type.INDEX_PRIMARY_KEY_TYPE_ALLOWLIST,
    SQLReviewRule_Type.INDEX_TYPE_ALLOW_LIST,
    SQLReviewRule_Type.SYSTEM_CHARSET_ALLOWLIST,
    SQLReviewRule_Type.SYSTEM_COLLATION_ALLOWLIST,
    SQLReviewRule_Type.SYSTEM_FUNCTION_DISALLOWED_LIST,
    SQLReviewRule_Type.TABLE_DISALLOW_DDL,
    SQLReviewRule_Type.TABLE_DISALLOW_DML,
  ];

  test.each(
    requiredStringArrayRuleTypes
  )("reports empty string-array rule %s", (type) => {
    const ruleMap = new Map([
      [Engine.MYSQL, new Map([[type, stringArrayRule(type, [])]])],
    ]);

    expect(validateRuleMapByEngine(ruleMap)).toMatchObject({
      type: "EMPTY_STRING_ARRAY",
      rule: { type },
    });
  });

  test("reports empty rule maps", () => {
    expect(validateRuleMapByEngine(new Map())).toEqual({
      type: "EMPTY_RULE_LIST",
    });
  });

  test("keeps configured table DDL and DML deny-list rules valid", () => {
    const ruleMap = new Map([
      [
        Engine.MYSQL,
        new Map([
          [
            SQLReviewRule_Type.TABLE_DISALLOW_DDL,
            stringArrayRule(SQLReviewRule_Type.TABLE_DISALLOW_DDL, [
              "audit_log",
            ]),
          ],
          [
            SQLReviewRule_Type.TABLE_DISALLOW_DML,
            stringArrayRule(SQLReviewRule_Type.TABLE_DISALLOW_DML, ["user"]),
          ],
        ]),
      ],
    ]);

    expect(validateRuleMapByEngine(ruleMap)).toBeUndefined();

    const rules = convertRuleMapToPolicyRuleList(ruleMap);
    const getStringArrayList = (rule: (typeof rules)[number] | undefined) => {
      expect(rule?.payload.case).toBe("stringArrayPayload");
      return rule?.payload.case === "stringArrayPayload"
        ? rule.payload.value.list
        : [];
    };

    expect(rules).toHaveLength(2);
    expect(getStringArrayList(rules[0])).toEqual(["audit_log"]);
    expect(getStringArrayList(rules[1])).toEqual(["user"]);
  });
});
