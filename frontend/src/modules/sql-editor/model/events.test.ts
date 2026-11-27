// @vitest-environment node
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
    const unsubscribe = sqlEditorEvents.on("append-editor-content", handler);

    await sqlEditorEvents.emit("append-editor-content", {
      content: "SELECT 1",
      select: true,
    });

    // emittery v2 wraps the payload in an envelope; every listener in the
    // app destructures `data`. If this shape changes again, callsites break.
    expect(handler).toHaveBeenCalledWith({
      name: "append-editor-content",
      data: { content: "SELECT 1", select: true },
    });
    unsubscribe();
  });
});
