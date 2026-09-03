import { describe, expect, test, vi } from "vitest";
import { planEvents } from "./events";

describe("planEvents", () => {
  test("delivers database change issue creation", async () => {
    const listener = vi.fn();
    const off = planEvents.on("database-change-issue-created", listener);

    await planEvents.emit("database-change-issue-created", {
      issue: "projects/app/issues/1",
      project: "projects/app",
    });

    expect(listener).toHaveBeenCalledWith({
      name: "database-change-issue-created",
      data: {
        issue: "projects/app/issues/1",
        project: "projects/app",
      },
    });
    off();
  });
});
