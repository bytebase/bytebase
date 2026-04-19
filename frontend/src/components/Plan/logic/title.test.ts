import { describe, expect, it, vi } from "vitest";
import { planQueryNameForProject } from "./title";

describe("planQueryNameForProject", () => {
  it("returns undefined when the project enforces manual title", () => {
    const result = planQueryNameForProject(
      { enforceIssueTitle: true },
      () => "Auto Title"
    );
    expect(result).toBeUndefined();
  });

  it("returns the generated title when the project does not enforce manual title", () => {
    const result = planQueryNameForProject(
      { enforceIssueTitle: false },
      () => "Auto Title"
    );
    expect(result).toBe("Auto Title");
  });

  it("does not invoke the generator when manual title is enforced", () => {
    const generate = vi.fn(() => "Auto Title");
    planQueryNameForProject({ enforceIssueTitle: true }, generate);
    expect(generate).not.toHaveBeenCalled();
  });

  it("invokes the generator exactly once when manual title is not enforced", () => {
    const generate = vi.fn(() => "Auto Title");
    planQueryNameForProject({ enforceIssueTitle: false }, generate);
    expect(generate).toHaveBeenCalledTimes(1);
  });
});
