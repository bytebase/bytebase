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

  test("listeners receive the emittery v2 {name, data} envelope", async () => {
    const handler = vi.fn();
    const unsubscribe = sqlEditorEvents.on("insert-at-caret", handler);

    await sqlEditorEvents.emit("insert-at-caret", { content: "SELECT 1" });

    // emittery v2 wraps the payload in an envelope; every listener in the
    // app destructures `data`. If this shape changes again, callsites break.
    expect(handler).toHaveBeenCalledWith({
      name: "insert-at-caret",
      data: { content: "SELECT 1" },
    });
    unsubscribe();
  });
});
