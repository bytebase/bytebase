import { describe, expect, test } from "vitest";

import source from "./PlanDetailRollbackSheet.tsx?raw";

const rollbackReadonlyMonacoProps = (src: string): string => {
  const match = src.match(
    /<ReadonlyMonaco[\s\S]*?content=\{preview\.statement\}[\s\S]*?\/>/
  );
  expect(match, "expected rollback preview ReadonlyMonaco").not.toBeNull();
  return match?.[0] ?? "";
};

describe("PlanDetailRollbackSheet statement preview", () => {
  test("lets Monaco own the compact preview viewport", () => {
    const props = rollbackReadonlyMonacoProps(source);

    expect(props).toContain("max={256}");
    expect(props).toContain("min={128}");
    expect(props).not.toContain("max-h-[320px]");
    expect(props).not.toContain("overflow-hidden");
  });
});
