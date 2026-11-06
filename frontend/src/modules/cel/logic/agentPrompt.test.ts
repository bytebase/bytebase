// @vitest-environment node
import { describe, expect, test } from "vitest";
import {
  CEL_ATTRIBUTE_ISSUE_LABELS,
  CEL_ATTRIBUTE_REQUEST_TIME,
  CEL_ATTRIBUTE_REQUEST_UNMASK,
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_DATABASE,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";
import type { Factor } from "../types/factor";
import type { Operator } from "../types/operator";
import { buildSelectorCapabilities } from "./agentPrompt";

describe("buildSelectorCapabilities", () => {
  test("describes direct table-name operations without transformed receivers or raw CEL", () => {
    expect(
      JSON.parse(buildSelectorCapabilities([CEL_ATTRIBUTE_RESOURCE_TABLE_NAME]))
    ).toEqual({
      fields: [
        {
          name: "resource.table_name",
          valueType: "string",
          templates: [
            'resource.table_name == "value"',
            'resource.table_name != "value"',
            'resource.table_name in ["value"]',
            '!(resource.table_name in ["value"])',
            'resource.table_name.contains("value")',
            '!(resource.table_name.contains("value"))',
            'resource.table_name.matches("value")',
            'resource.table_name.startsWith("value")',
            'resource.table_name.endsWith("value")',
          ],
        },
      ],
      grouping: ["&&", "||", "parentheses"],
      unsupported: [
        "transformed-receivers",
        "arbitrary-functions",
        "raw-cel",
        "general-negation",
      ],
    });
  });

  test("uses the field value types and literal shapes supported by the selector", () => {
    const capabilities = JSON.parse(
      buildSelectorCapabilities([
        CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
        CEL_ATTRIBUTE_REQUEST_UNMASK,
        CEL_ATTRIBUTE_REQUEST_TIME,
      ])
    );

    expect(capabilities.fields).toEqual([
      {
        name: "resource.classification_level",
        valueType: "integer",
        templates: [
          "resource.classification_level == 0",
          "resource.classification_level != 0",
          "resource.classification_level < 0",
          "resource.classification_level <= 0",
          "resource.classification_level >= 0",
          "resource.classification_level > 0",
          "resource.classification_level in [0]",
          "!(resource.classification_level in [0])",
        ],
      },
      {
        name: "request.unmask",
        valueType: "boolean",
        templates: ["request.unmask == true", "request.unmask != true"],
      },
      { name: "request.time", valueType: "timestamp", templates: [] },
    ]);
  });

  test("only advertises operations permitted by each field's override map", () => {
    const overrideMap = new Map<Factor, Operator[]>([
      [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, ["matches"]],
      [CEL_ATTRIBUTE_RESOURCE_DATABASE, ["_==_", "@in"]],
    ]);

    expect(
      JSON.parse(
        buildSelectorCapabilities(
          [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, CEL_ATTRIBUTE_RESOURCE_DATABASE],
          overrideMap
        )
      ).fields
    ).toEqual([
      {
        name: "resource.table_name",
        valueType: "string",
        templates: ['resource.table_name.matches("value")'],
      },
      {
        name: "resource.database",
        valueType: "string",
        templates: [
          'resource.database == "value"',
          'resource.database in ["value"]',
        ],
      },
    ]);
  });

  test("preserves an explicit empty operator override", () => {
    const overrideMap = new Map<Factor, Operator[]>([
      [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME, []],
    ]);

    expect(
      JSON.parse(
        buildSelectorCapabilities(
          [CEL_ATTRIBUTE_RESOURCE_TABLE_NAME],
          overrideMap
        )
      ).fields
    ).toEqual([
      { name: "resource.table_name", valueType: "string", templates: [] },
    ]);
  });

  test("expresses list membership and negated operations as valid CEL rather than internal operator names", () => {
    const capabilities = JSON.parse(
      buildSelectorCapabilities([CEL_ATTRIBUTE_ISSUE_LABELS])
    );

    expect(capabilities.fields).toEqual([
      {
        name: "issue.labels",
        valueType: "string[]",
        templates: ['"value" in issue.labels', '!("value" in issue.labels)'],
      },
    ]);
  });

  test("does not advertise arbitrary negation, grouping, or incompatible field operations as predicates", () => {
    const overrideMap = new Map<Factor, Operator[]>([
      [CEL_ATTRIBUTE_REQUEST_UNMASK, ["contains", "!_", "_&&_"]],
      [CEL_ATTRIBUTE_REQUEST_TIME, ["_>_"]],
    ]);

    expect(
      JSON.parse(
        buildSelectorCapabilities(
          [CEL_ATTRIBUTE_REQUEST_UNMASK, CEL_ATTRIBUTE_REQUEST_TIME],
          overrideMap
        )
      ).fields
    ).toEqual([
      { name: "request.unmask", valueType: "boolean", templates: [] },
      { name: "request.time", valueType: "timestamp", templates: [] },
    ]);
  });

  test("does not invent available fields when the editor has none", () => {
    expect(JSON.parse(buildSelectorCapabilities([])).fields).toEqual([]);
  });
});
