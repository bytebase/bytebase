// @vitest-environment node
import { describe, expect, test, vi } from "vitest";
import type { AppRouterInstance } from "@/app/router";

vi.mock("../../dom", () => ({
  lazyExecuteDomAction: vi.fn(async () => ({ success: true, message: "done" })),
}));

import { lazyExecuteDomAction } from "../../dom";
import { createDomActionTool } from "./domAction";

describe("createDomActionTool", () => {
  test("forwards read actions with ref-based args", async () => {
    const router = { push: vi.fn() } as unknown as AppRouterInstance;
    const tool = createDomActionTool(router);

    const result = await tool({ type: "read", ref: "e3" });

    expect(lazyExecuteDomAction).toHaveBeenCalledWith(
      { type: "read", ref: "e3" },
      router,
      undefined
    );
    expect(result).toBe(JSON.stringify({ success: true, message: "done" }));
  });

  test("forwards the current-page predicate", async () => {
    const router = { push: vi.fn() } as unknown as AppRouterInstance;
    const canUseCurrentPage = vi.fn(() => false);
    const tool = createDomActionTool(router, canUseCurrentPage);

    await tool({ type: "click", ref: "e1" });

    expect(lazyExecuteDomAction).toHaveBeenCalledWith(
      { type: "click", ref: "e1" },
      router,
      canUseCurrentPage
    );
  });
});
