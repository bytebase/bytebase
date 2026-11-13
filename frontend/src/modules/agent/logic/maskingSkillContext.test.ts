import { describe, expect, test } from "vitest";
import {
  getMaskingSectionsForPage,
  getMaskingSectionsForTurn,
  isMaskingToolEnabled,
} from "./maskingSkillContext";
import type { Message } from "./types";

describe("masking skill context", () => {
  test("seeds only the page's relevant workflow", () => {
    expect(getMaskingSectionsForPage()).toEqual([]);
    expect(getMaskingSectionsForPage({ kind: "global-masking" })).toEqual([
      "overview",
      "rule",
    ]);
    expect(getMaskingSectionsForPage({ kind: "semantic-types" })).toEqual([
      "overview",
      "semantic-type",
    ]);
    expect(
      getMaskingSectionsForPage({
        kind: "masking-exemption",
        project: "projects/example",
      })
    ).toEqual(["overview", "exemption"]);
    expect(getMaskingSectionsForPage({ kind: "data-classification" })).toEqual([
      "overview",
    ]);
  });

  test("includes overview dependencies without exposing unrelated drafts", () => {
    expect(isMaskingToolEnabled("get_masking_overview", ["rule"])).toBe(true);
    expect(isMaskingToolEnabled("prepare_masking_rule_draft", [])).toBe(false);
    expect(isMaskingToolEnabled("prepare_masking_rule_draft", ["rule"])).toBe(
      true
    );
    expect(
      isMaskingToolEnabled("prepare_masking_exemption_draft", ["rule"])
    ).toBe(false);
    expect(isMaskingToolEnabled("get_database_count", [])).toBe(true);
    expect(
      isMaskingToolEnabled("ask_masking_semantic_type", ["exemption"])
    ).toBe(true);
    expect(isMaskingToolEnabled("ask_unmask_method", [])).toBe(false);
    expect(isMaskingToolEnabled("ask_unmask_method", ["overview"])).toBe(false);
    expect(isMaskingToolEnabled("ask_unmask_method", ["rule"])).toBe(false);
    expect(isMaskingToolEnabled("ask_unmask_method", ["semantic-type"])).toBe(
      false
    );
    expect(isMaskingToolEnabled("ask_unmask_method", ["exemption"])).toBe(true);
  });

  const loaded: Message[] = [
    { role: "user", content: "Help with masking" },
    {
      role: "assistant",
      toolCalls: [
        {
          id: "skill",
          name: "get_skill",
          arguments: JSON.stringify({ name: "data-masking", section: "rule" }),
        },
      ],
    },
    {
      role: "tool",
      toolCallId: "skill",
      content: JSON.stringify({
        kind: "loaded",
        name: "data-masking",
        section: "rule",
      }),
    },
  ];

  test("restores registered successful loads only within the current turn", () => {
    expect(getMaskingSectionsForTurn(loaded)).toEqual(["overview", "rule"]);
    expect(
      getMaskingSectionsForTurn([
        ...loaded,
        { role: "user", content: "Review this Plan" },
      ])
    ).toEqual([]);
    expect(getMaskingSectionsForTurn([loaded[0]!, loaded[2]!])).toEqual([]);
    expect(
      getMaskingSectionsForTurn([
        loaded[0]!,
        loaded[1]!,
        {
          ...loaded[2]!,
          content: JSON.stringify({
            kind: "error",
            name: "data-masking",
            section: "rule",
          }),
        },
      ])
    ).toEqual([]);
    expect(
      getMaskingSectionsForTurn([
        loaded[0]!,
        {
          ...loaded[1]!,
          toolCalls: [
            {
              id: "skill",
              name: "get_skill",
              arguments: JSON.stringify({
                name: "data-masking",
                section: "unknown",
              }),
            },
          ],
        },
        loaded[2]!,
      ])
    ).toEqual([]);
  });

  test("restores legacy masking clarification capability without approval authority", () => {
    const messages: Message[] = [
      { role: "user", content: "Mask these columns" },
      {
        role: "assistant",
        toolCalls: [
          {
            id: "overview",
            name: "get_masking_overview",
            arguments: JSON.stringify({ project: "projects/example" }),
          },
        ],
      },
      {
        role: "tool",
        toolCallId: "overview",
        content: JSON.stringify({ project: "projects/example" }),
      },
      {
        role: "assistant",
        toolCalls: [
          {
            id: "scope",
            name: "ask_masking_rule_scope",
            arguments: JSON.stringify({ project: "projects/example" }),
          },
        ],
      },
      {
        role: "tool",
        toolCallId: "scope",
        content: JSON.stringify({
          kind: "choose",
          answer: "This project",
          value: "projects/example",
        }),
      },
    ];
    expect(getMaskingSectionsForTurn(messages)).toEqual(["overview", "rule"]);
  });

  test.each(["masking-exemption", "just-in-time"])(
    "restores only exemption context for a legacy unmask method choice %s",
    (value) => {
      const messages: Message[] = [
        { role: "user", content: "Help me view unmasked data" },
        {
          role: "assistant",
          toolCalls: [
            { id: "method", name: "ask_unmask_method", arguments: "{}" },
          ],
        },
        {
          role: "tool",
          toolCallId: "method",
          content: JSON.stringify({
            kind: "choose",
            answer: "Selected method",
            value,
          }),
        },
      ];
      expect(getMaskingSectionsForTurn(messages)).toEqual([
        "overview",
        "exemption",
      ]);
    }
  );

  test("preserves a loaded exemption section after an unmask method choice", () => {
    const messages: Message[] = [
      loaded[0]!,
      {
        role: "assistant",
        toolCalls: [
          {
            id: "skill",
            name: "get_skill",
            arguments: JSON.stringify({
              name: "data-masking",
              section: "exemption",
            }),
          },
        ],
      },
      {
        role: "tool",
        toolCallId: "skill",
        content: JSON.stringify({
          kind: "loaded",
          name: "data-masking",
          section: "exemption",
        }),
      },
      {
        role: "assistant",
        toolCalls: [
          {
            id: "method",
            name: "ask_unmask_method",
            arguments: JSON.stringify({ project: "projects/orders" }),
          },
        ],
      },
      {
        role: "tool",
        toolCallId: "method",
        content: JSON.stringify({
          kind: "choose",
          answer: "Masking exemption",
          value: "masking-exemption",
        }),
      },
    ];
    const before = structuredClone(messages);
    expect(getMaskingSectionsForTurn(messages)).toEqual([
      "overview",
      "exemption",
    ]);
    expect(messages).toEqual(before);
  });

  test("does not restore an acknowledgement with an ambiguous call ID", () => {
    expect(getMaskingSectionsForTurn([...loaded, loaded[1]!])).toEqual([]);
  });
});
