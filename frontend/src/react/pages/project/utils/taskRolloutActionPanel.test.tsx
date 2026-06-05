import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  ScheduledRunTimeInput,
  TASK_ROLLOUT_ACTION_SHEET_WIDTH,
} from "./taskRolloutActionPanel";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("@/react/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => (
    <input {...props} />
  ),
}));

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
  it("renders a compact datetime input", () => {
    act(() => {
      root.render(
        <ScheduledRunTimeInput
          onChange={() => {}}
          placeholder="Select scheduled time"
          value={Date.UTC(2026, 5, 4, 9, 30)}
        />
      );
    });

    const input = container.querySelector(
      'input[type="datetime-local"]'
    ) as HTMLInputElement;

    expect(input.className).toContain("w-64");
    expect(input.className).toContain("max-w-full");
  });
});

describe("TASK_ROLLOUT_ACTION_SHEET_WIDTH", () => {
  it("uses the standard drawer width for rollout task actions", () => {
    expect(TASK_ROLLOUT_ACTION_SHEET_WIDTH).toBe("standard");
  });
});
