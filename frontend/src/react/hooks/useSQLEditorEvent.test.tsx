import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { sqlEditorEvents } from "@/views/sql-editor/events";
import { useSQLEditorEvent } from "./useSQLEditorEvent";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

describe("useSQLEditorEvent", () => {
  let root: Root | undefined;
  let container: HTMLDivElement | undefined;

  afterEach(() => {
    act(() => {
      root?.unmount();
    });
    root = undefined;
    container = undefined;
    document.body.innerHTML = "";
  });

  test("handler called when event emitted after subscribe", async () => {
    const handler = vi.fn();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    function Harness() {
      useSQLEditorEvent("format-content", handler);
      return null;
    }

    await act(async () => {
      root!.render(<Harness />);
    });

    await act(async () => {
      await sqlEditorEvents.emit("format-content", undefined);
    });

    expect(handler).toHaveBeenCalledTimes(1);
  });

  test("handler NOT called after component unmounts (unsubscribe works)", async () => {
    const handler = vi.fn();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    function Harness() {
      useSQLEditorEvent("format-content", handler);
      return null;
    }

    await act(async () => {
      root!.render(<Harness />);
    });

    await act(async () => {
      root!.unmount();
    });
    root = undefined;

    await sqlEditorEvents.emit("format-content", undefined);

    expect(handler).not.toHaveBeenCalled();
  });

  test("handler ref updates: rerender with new handler, emit, latest handler fires", async () => {
    const handler1 = vi.fn();
    const handler2 = vi.fn();
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);

    function Harness({ onEvent }: { onEvent: () => void }) {
      useSQLEditorEvent("format-content", onEvent);
      return null;
    }

    await act(async () => {
      root!.render(<Harness onEvent={handler1} />);
    });

    // Rerender with new handler
    await act(async () => {
      root!.render(<Harness onEvent={handler2} />);
    });

    await act(async () => {
      await sqlEditorEvents.emit("format-content", undefined);
    });

    expect(handler1).not.toHaveBeenCalled();
    expect(handler2).toHaveBeenCalledTimes(1);
  });
});
