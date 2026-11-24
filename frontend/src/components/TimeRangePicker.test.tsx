import {
  cleanup,
  fireEvent,
  render,
  screen,
  within,
} from "@testing-library/react";
import { useState } from "react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { SearchParams } from "./AdvancedSearch";
import { TimeRangePicker } from "./TimeRangePicker";
import { formatRangeScope, quickRange } from "./time-range";

const mocks = vi.hoisted(() => ({ compact: false }));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useMediaQuery", () => ({
  useMediaQuery: () => mocks.compact,
}));

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
}
globalThis.ResizeObserver =
  ResizeObserverMock as unknown as typeof ResizeObserver;

const NOW = new Date(2026, 10, 24, 10, 42);
const local = (...args: [number, number, number, number?, number?, number?, number?]) => {
  const [year, month, day, hour = 0, minute = 0, second = 0, ms = 0] = args;
  return new Date(year, month - 1, day, hour, minute, second, ms).getTime();
};
const endOfDay = (year: number, month: number, day: number) =>
  local(year, month, day, 23, 59, 59, 999);

const onParamsChange = vi.fn<(params: SearchParams) => void>();

function Harness({ initial }: { initial: SearchParams }) {
  const [params, setParams] = useState(initial);
  return (
    <TimeRangePicker
      label="Created"
      params={params}
      onParamsChange={(next) => {
        onParamsChange(next);
        setParams(next);
      }}
    />
  );
}

const renderPicker = (created?: string) =>
  render(
    <Harness
      initial={{
        query: "",
        scopes: [
          { id: "status", value: "OPEN" },
          ...(created ? [{ id: "created", value: created }] : []),
        ],
      }}
    />
  );

const trigger = () => screen.getByRole("button", { name: /^Created/ });
const openPicker = () => fireEvent.click(trigger());
const pickDay = (day: number) =>
  fireEvent.click(
    within(screen.getByRole("grid")).getByRole("button", {
      name: new RegExp(`\\b${day}(st|nd|rd|th)\\b`),
    })
  );
const dayButton = (end: "from" | "to") =>
  screen.getByRole("button", { name: new RegExp(`^common\\.${end}:`) });
const applyButton = () => screen.getByRole("button", { name: "common.apply" });
const timeField = (end: "from" | "to") =>
  screen.getByRole("textbox", { name: `time-range.${end}-time` });
const typeTime = (end: "from" | "to", value: string) => {
  fireEvent.change(timeField(end), { target: { value } });
  fireEvent.blur(timeField(end));
};
const lastCreated = () =>
  onParamsChange.mock.lastCall?.[0].scopes.find((s) => s.id === "created");

beforeEach(() => {
  mocks.compact = false;
  onParamsChange.mockClear();
  vi.useFakeTimers({ toFake: ["Date"] });
  vi.setSystemTime(NOW);
});

afterEach(() => {
  cleanup();
  vi.useRealTimers();
});

describe("TimeRangePicker", () => {
  test("a quick range applies on click and names the trigger", () => {
    renderPicker();
    openPicker();
    fireEvent.click(
      screen.getByRole("button", { name: "time-range.last-7-days" })
    );

    expect(onParamsChange).toHaveBeenCalledWith({
      query: "",
      scopes: [
        { id: "status", value: "OPEN" },
        {
          id: "created",
          value: `${local(2026, 11, 18)},${endOfDay(2026, 11, 24)}`,
          readonly: true,
        },
      ],
    });
    expect(screen.queryByRole("grid")).toBeNull();
    expect(trigger().textContent).toContain("time-range.last-7-days");
  });

  test("two picked days apply as whole days through the end of the last one", () => {
    renderPicker();
    openPicker();
    pickDay(15);
    pickDay(20);
    fireEvent.click(applyButton());

    expect(lastCreated()?.value).toBe(
      `${local(2026, 11, 15)},${endOfDay(2026, 11, 20)}`
    );
    expect(trigger().textContent).toContain("2026-11-15");
    expect(trigger().textContent).toContain("2026-11-20");
    expect(trigger().textContent).not.toContain("23:59");
  });

  test("the active end follows each pick, and a day before the start restarts the range", () => {
    renderPicker();
    openPicker();
    expect(dayButton("from").getAttribute("aria-pressed")).toBe("true");

    pickDay(20);
    expect(dayButton("to").getAttribute("aria-pressed")).toBe("true");
    expect(applyButton()).toHaveProperty("disabled", true);

    pickDay(15);
    expect(dayButton("from").textContent).toBe("2026-11-15");
    expect(dayButton("to").textContent).toBe("time-range.end-day");
    expect(dayButton("to").getAttribute("aria-pressed")).toBe("true");
  });

  test("a typed time narrows the range and shows on the trigger", () => {
    renderPicker();
    openPicker();
    pickDay(15);
    pickDay(20);
    typeTime("from", "930");
    expect(timeField("from")).toHaveProperty("value", "09:30");
    fireEvent.click(applyButton());

    expect(lastCreated()?.value).toBe(
      `${local(2026, 11, 15, 9, 30)},${endOfDay(2026, 11, 20)}`
    );
    expect(trigger().textContent).toContain("2026-11-15 09:30");
  });

  test("an unreadable time is reported and blocks Apply until fixed", () => {
    renderPicker();
    openPicker();
    pickDay(15);
    pickDay(20);
    typeTime("to", "25");

    expect(screen.getByRole("alert").textContent).toBe(
      "time-range.invalid-time"
    );
    expect(timeField("to").getAttribute("aria-invalid")).toBe("true");
    expect(applyButton()).toHaveProperty("disabled", true);

    fireEvent.change(timeField("to"), { target: { value: "18:00" } });
    expect(screen.queryByRole("alert")).toBeNull();
    expect(applyButton()).toHaveProperty("disabled", false);
  });

  test("an end before the start on the same day is reported", () => {
    renderPicker();
    openPicker();
    pickDay(20);
    pickDay(20);
    typeTime("from", "18:00");
    typeTime("to", "09:00");

    expect(screen.getByRole("alert").textContent).toBe(
      "time-range.to-before-from"
    );
    expect(applyButton()).toHaveProperty("disabled", true);
  });

  test("the month arrows page the calendar without losing the draft", () => {
    renderPicker();
    openPicker();
    pickDay(15);
    expect(screen.getByRole("grid").getAttribute("aria-label")).toContain(
      "November 2026"
    );

    fireEvent.click(screen.getByRole("button", { name: /next month/i }));
    expect(screen.getByRole("grid").getAttribute("aria-label")).toContain(
      "December 2026"
    );
    fireEvent.click(screen.getByRole("button", { name: /previous month/i }));
    fireEvent.click(screen.getByRole("button", { name: /previous month/i }));
    expect(screen.getByRole("grid").getAttribute("aria-label")).toContain(
      "October 2026"
    );
    expect(dayButton("from").textContent).toBe("2026-11-15");
  });

  test("All time is marked while no range is set, and picking it removes the range", () => {
    renderPicker();
    openPicker();
    const allTime = () =>
      screen.getByRole("button", { name: "time-range.all-time" });
    expect(allTime().getAttribute("aria-pressed")).toBe("true");

    pickDay(15);
    expect(allTime().getAttribute("aria-pressed")).toBe("false");
    pickDay(20);
    fireEvent.click(applyButton());
    expect(lastCreated()).toBeDefined();

    openPicker();
    fireEvent.click(allTime());
    expect(onParamsChange).toHaveBeenLastCalledWith({
      query: "",
      scopes: [{ id: "status", value: "OPEN" }],
    });
    expect(screen.queryByRole("grid")).toBeNull();
  });

  test("the trigger's clear button removes only the range", () => {
    renderPicker("1000,2000");
    fireEvent.click(screen.getByRole("button", { name: "time-range.clear" }));

    expect(onParamsChange).toHaveBeenCalledWith({
      query: "",
      scopes: [{ id: "status", value: "OPEN" }],
    });
    expect(screen.queryByRole("button", { name: "time-range.clear" })).toBeNull();
  });

  test("reopening starts from the applied range, not an abandoned draft", () => {
    renderPicker();
    openPicker();
    pickDay(15);
    fireEvent.keyDown(document.activeElement ?? document.body, {
      key: "Escape",
    });
    expect(screen.queryByRole("grid")).toBeNull();

    openPicker();
    expect(dayButton("from").textContent).toBe("time-range.start-day");
    expect(onParamsChange).not.toHaveBeenCalled();
  });

  test("a stored range is named only while it still equals a quick range", () => {
    const yesterday = new Date(2026, 10, 23, 10, 42);
    renderPicker(formatRangeScope(quickRange("last-7-days", yesterday)));

    expect(trigger().textContent).not.toContain("time-range.last-7-days");
    expect(trigger().textContent).toContain("2026-11-17");
    expect(trigger().textContent).toContain("2026-11-23");
  });

  describe("compact layout", () => {
    beforeEach(() => {
      mocks.compact = true;
    });

    test("keeps time fields out of the way until asked for", () => {
      renderPicker();
      openPicker();
      expect(screen.queryAllByRole("textbox")).toHaveLength(0);

      fireEvent.click(
        screen.getByRole("button", { name: "time-range.add-time" })
      );
      pickDay(15);
      pickDay(20);
      typeTime("from", "9");
      fireEvent.click(
        screen.getByRole("button", { name: "time-range.remove-time" })
      );
      expect(screen.queryAllByRole("textbox")).toHaveLength(0);
      fireEvent.click(applyButton());

      expect(lastCreated()?.value).toBe(
        `${local(2026, 11, 15)},${endOfDay(2026, 11, 20)}`
      );
    });

    test("opens with the time fields when the applied range has times", () => {
      renderPicker(`${local(2026, 11, 15, 9, 30)},${endOfDay(2026, 11, 20)}`);
      openPicker();

      expect(timeField("from")).toHaveProperty("value", "09:30");
      expect(timeField("to")).toHaveProperty("value", "23:59");
    });
  });
});
