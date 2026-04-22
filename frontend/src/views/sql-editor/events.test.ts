import Emittery from "emittery";
import { describe, expect, test, vi } from "vitest";
import { sqlEditorEvents } from "./events";

describe("sqlEditorEvents", () => {
  test("singleton exists and is an Emittery instance", () => {
    expect(sqlEditorEvents).toBeInstanceOf(Emittery);
  });

  test("emit + on roundtrip works", async () => {
    const handler = vi.fn();
    const unsubscribe = sqlEditorEvents.on("format-content", handler);

    await sqlEditorEvents.emit("format-content", undefined);

    expect(handler).toHaveBeenCalledTimes(1);
    unsubscribe();
  });
});
