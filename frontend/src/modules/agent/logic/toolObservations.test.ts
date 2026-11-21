// @vitest-environment node
import { describe, expect, test } from "vitest";
import { getCurrentTurnToolCalls } from "./toolObservations";
import type { Message } from "./types";

const call = (id: string, operationId: string): Message => ({
  role: "assistant",
  toolCalls: [
    { id, name: "call_api", arguments: JSON.stringify({ operationId }) },
  ],
});
const result = (id: string, content: string): Message => ({
  role: "tool",
  toolCallId: id,
  content,
});

describe("getCurrentTurnToolCalls", () => {
  test("associates reused IDs with their own occurrence", () => {
    expect(
      getCurrentTurnToolCalls([
        { role: "user", content: "Check settings" },
        call("same", "SettingService/UpdateSetting"),
        result("same", "first failure"),
        call("same", "DatabaseService/GetDatabase"),
        result("same", "second success"),
      ])
    ).toEqual([
      {
        name: "call_api",
        args: { operationId: "SettingService/UpdateSetting" },
        result: "first failure",
      },
      {
        name: "call_api",
        args: { operationId: "DatabaseService/GetDatabase" },
        result: "second success",
      },
    ]);
  });

  test("ignores prior turns and orphaned duplicate results", () => {
    expect(
      getCurrentTurnToolCalls([
        { role: "user", content: "Earlier" },
        call("old", "SettingService/UpdateSetting"),
        result("old", "old failure"),
        { role: "user", content: "Now" },
        result("old", "orphan"),
        call("new", "DatabaseService/GetDatabase"),
        result("new", "success"),
        result("new", "late duplicate failure"),
      ])
    ).toEqual([
      {
        name: "call_api",
        args: { operationId: "DatabaseService/GetDatabase" },
        result: "success",
      },
    ]);
  });

  test("does not attach a later result to an abandoned earlier batch", () => {
    const observations = getCurrentTurnToolCalls([
      { role: "user", content: "Check" },
      call("abandoned", "SettingService/UpdateSetting"),
      call("current", "DatabaseService/GetDatabase"),
      result("abandoned", "late failure"),
      result("current", "success"),
    ]);
    expect(observations[0]).not.toHaveProperty("result");
    expect(observations[1].result).toBe("success");
  });
});
