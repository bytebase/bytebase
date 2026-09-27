import dayjs from "dayjs";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it } from "vitest";
import {
  ScheduledRunTimeInput,
  TASK_ROLLOUT_ACTION_SHEET_WIDTH,
} from "./taskRolloutActionPanel";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

let container: HTMLDivElement;
let root: ReturnType<typeof createRoot>;

beforeEach(() => {
  container = document.createElement("div");
  document.body.appendChild(container);
  root = createRoot(container);
});

afterEach(() => {
  act(() => {
    root.unmount();
  });
  document.body.removeChild(container);
});

describe("ScheduledRunTimeInput", () => {
  it("renders a picker trigger with the formatted value, not a native input", () => {
    const value = new Date(2026, 5, 4, 9, 30).getTime();
    act(() => {
      root.render(
        <ScheduledRunTimeInput
          onChange={() => {}}
          placeholder="Select scheduled time"
          value={value}
        />
      );
    });

    // No native datetime-local input — it's a styled popover trigger.
    expect(container.querySelector('input[type="datetime-local"]')).toBeNull();

    const trigger = container.querySelector("button") as HTMLButtonElement;
    expect(trigger).not.toBeNull();
    expect(trigger.textContent).toContain(
      dayjs(value).format("YYYY-MM-DD HH:mm")
    );
  });

  it("shows the placeholder when no value is set", () => {
    act(() => {
      root.render(
        <ScheduledRunTimeInput
          onChange={() => {}}
          placeholder="Select scheduled time"
          value={undefined}
        />
      );
    });

    const trigger = container.querySelector("button") as HTMLButtonElement;
    expect(trigger.textContent).toContain("Select scheduled time");
  });
});

describe("TASK_ROLLOUT_ACTION_SHEET_WIDTH", () => {
  it("uses the standard drawer width for rollout task actions", () => {
    expect(TASK_ROLLOUT_ACTION_SHEET_WIDTH).toBe("standard");
  });
});
