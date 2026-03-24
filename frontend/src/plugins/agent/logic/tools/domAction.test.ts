import { describe, expect, test, vi } from "vitest";
import type { Router } from "vue-router";

vi.mock("../../dom", () => ({
  lazyExecuteDomAction: vi.fn(async () => ({ success: true, message: "done" })),
}));

import { lazyExecuteDomAction } from "../../dom";
import { createDomActionTool } from "./domAction";

describe("createDomActionTool", () => {
  test("forwards read actions with ref-based args", async () => {
    const router = { push: vi.fn() } as unknown as Router;
    const tool = createDomActionTool(router);

    const result = await tool({ type: "read", ref: "e3" });

    expect(lazyExecuteDomAction).toHaveBeenCalledWith(
      { type: "read", ref: "e3" },
      router
    );
    expect(result).toBe(JSON.stringify({ success: true, message: "done" }));
  });
});
