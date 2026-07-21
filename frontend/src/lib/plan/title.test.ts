import { describe, expect, it, vi } from "vitest";
import { applyPlanTitleToQuery, planQueryNameForProject } from "./title";

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

describe("applyPlanTitleToQuery", () => {
  it("does not set query.name when the project enforces manual title", () => {
    const query: Record<string, string> = {
      template: "bb.plan.change-database",
    };
    applyPlanTitleToQuery(
      query,
      { enforceIssueTitle: true },
      () => "Auto Title"
    );
    expect(query).toEqual({ template: "bb.plan.change-database" });
    expect(query.name).toBeUndefined();
  });

  it("sets query.name to the generator output when manual title is not enforced", () => {
    const query: Record<string, string> = {};
    applyPlanTitleToQuery(
      query,
      { enforceIssueTitle: false },
      () => "Auto Title"
    );
    expect(query.name).toBe("Auto Title");
  });

  it("does not invoke the generator when manual title is enforced", () => {
    const generate = vi.fn(() => "Auto Title");
    const query: Record<string, string> = {};
    applyPlanTitleToQuery(query, { enforceIssueTitle: true }, generate);
    expect(generate).not.toHaveBeenCalled();
  });

  it("preserves pre-existing keys on the query", () => {
    const query: Record<string, string> = {
      template: "bb.plan.change-database",
      databaseList: "db1,db2",
    };
    applyPlanTitleToQuery(
      query,
      { enforceIssueTitle: false },
      () => "Auto Title"
    );
    expect(query).toEqual({
      template: "bb.plan.change-database",
      databaseList: "db1,db2",
      name: "Auto Title",
    });
  });

  it("clears a pre-existing query.name when enforceIssueTitle is true", () => {
    const query: Record<string, string> = {
      template: "bb.plan.change-database",
      name: "stale auto title",
    };
    applyPlanTitleToQuery(query, { enforceIssueTitle: true }, () => "fresh");
    expect(query.name).toBeUndefined();
    expect(query.template).toBe("bb.plan.change-database"); // untouched
  });
});
