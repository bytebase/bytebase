import { describe, expect, test } from "vitest";

import source from "./DeployTaskItem.tsx?raw";

const statementReadonlyMonacoProps = (src: string): string => {
  const match = src.match(
    /<ReadonlyMonaco[\s\S]*?content=\{statement\}[\s\S]*?\/>/
  );
  expect(match, "expected statement ReadonlyMonaco").not.toBeNull();
  return match?.[0] ?? "";
};

describe("DeployTaskItem statement editor", () => {
  test("lets Monaco own the compact statement viewport", () => {
    const props = statementReadonlyMonacoProps(source);

    expect(props).toContain("max={256}");
    expect(props).toContain("min={128}");
    expect(props).not.toContain("max-h-64");
    expect(props).not.toContain("overflow-hidden");
  });
});
