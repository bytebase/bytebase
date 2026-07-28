import { beforeEach, describe, expect, test, vi } from "vitest";

const { addMock } = vi.hoisted(() => ({ addMock: vi.fn() }));
vi.mock("@base-ui/react/toast", () => ({
  Toast: {
    createToastManager: () => ({ add: addMock, close: vi.fn() }),
  },
}));

import { mapNotificationToToast, pushReactNotification } from "./toast";

describe("mapNotificationToToast", () => {
  test("SUCCESS maps to type=success, priority=low, timeout=6000", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "SUCCESS",
        title: "Saved",
      })
    ).toMatchObject({
      type: "success",
      priority: "low",
      timeout: 6000,
      title: "Saved",
    });
  });

  test("CRITICAL maps to type=error, priority=high, timeout=10000", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "CRITICAL",
        title: "Boom",
      })
    ).toMatchObject({
      type: "error",
      priority: "high",
      timeout: 10000,
    });
  });

  test("WARN maps to type=warning, INFO maps to type=info", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "WARN",
        title: "Heads up",
      })
    ).toMatchObject({ type: "warning" });
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "INFO",
        title: "FYI",
      })
    ).toMatchObject({ type: "info" });
  });

  test("manualHide=true sets timeout=0 (manager treats 0 as manual)", () => {
    expect(
      mapNotificationToToast({
        module: "bytebase",
        style: "INFO",
        title: "T",
        manualHide: true,
      })
    ).toMatchObject({ timeout: 0 });
  });

  test("description string passes through; link/linkTitle become actionProps", () => {
    const mapped = mapNotificationToToast({
      module: "bytebase",
      style: "INFO",
      title: "T",
      description: "details",
      link: "https://example.com",
      linkTitle: "Open",
    });
    expect(mapped.description).toBe("details");
    expect(mapped.actionProps).toMatchObject({
      "aria-label": "Open",
      onClick: expect.any(Function),
    });
  });
});

describe("pushReactNotification", () => {
  beforeEach(() => addMock.mockReset());

  test("calls toastManager.add with mapped options", () => {
    pushReactNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: "Saved",
    });
    expect(addMock).toHaveBeenCalledTimes(1);
    expect(addMock.mock.calls[0][0]).toMatchObject({
      title: "Saved",
      type: "success",
      priority: "low",
      timeout: 6000,
    });
  });

  test("ignores notifications with module !== 'bytebase'", () => {
    pushReactNotification({
      module: "other",
      style: "INFO",
      title: "ignored",
    });
    expect(addMock).not.toHaveBeenCalled();
  });
});
