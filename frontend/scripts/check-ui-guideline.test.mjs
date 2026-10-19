import { describe, expect, test } from "vitest";
import { scanSource } from "./check-ui-guideline.mjs";

describe("check-ui-guideline", () => {
  test("rejects a hand-built text pill", () => {
    expect(
      scanSource(
        'export const Status = () => <span className="rounded-full bg-success/10 px-1.5 py-0.5 text-xs">Active</span>;',
        "src/Status.tsx"
      )
    ).toEqual([
      {
        path: "src/Status.tsx",
        rule: "no-text-pill",
        token: "rounded-full",
        count: 1,
      },
    ]);
  });

  test("allows circular controls", () => {
    expect(
      scanSource(
        'export const Avatar = () => <span className="size-6 rounded-full bg-accent" />;',
        "src/Avatar.tsx"
      )
    ).toEqual([]);
  });

  test("rejects a Badge that overrides the compact corner", () => {
    expect(
      scanSource(
        'export const Status = () => <Badge className="rounded-full px-1.5 py-0.5 text-xs">Active</Badge>;',
        "src/Status.tsx"
      )
    ).toEqual([
      {
        path: "src/Status.tsx",
        rule: "no-text-pill",
        token: "rounded-full",
        count: 1,
      },
    ]);
  });

  test("allows the lifecycle status stamp pill", () => {
    expect(
      scanSource(
        'export const Status = () => <span className="rounded-full border px-2 py-0.5 text-sm">Deployed</span>;',
        "src/routes/project/plan-detail/components/lifecycle/LifecycleStamp.tsx"
      )
    ).toEqual([]);
  });
});
