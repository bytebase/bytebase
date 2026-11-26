// @vitest-environment node
import { describe, expect, test } from "vitest";
import { buildPlanFilter, buildPlanFindBySearchParams } from "./plan";

const QUOTED = 'SELECT * FROM "users"';
const ESCAPED = 'SELECT * FROM \\"users\\"';

describe("buildPlanFilter", () => {
  test("preserves milliseconds in both created-time bounds", () => {
    expect(
      buildPlanFilter({
        project: "projects/p",
        createdTsAfter: Date.UTC(2026, 10, 24, 0, 0, 0, 125),
        createdTsBefore: Date.UTC(2026, 10, 24, 23, 59, 59, 999),
      })
    ).toBe(
      'create_time >= "2026-11-24T00:00:00.125Z" && create_time <= "2026-11-24T23:59:59.999Z"'
    );
  });

  test("escapes a quote in the free-text query", () => {
    const filter = buildPlanFilter({
      project: "projects/p",
      query: QUOTED,
    }).toLowerCase();
    expect(filter).not.toContain(QUOTED.toLowerCase());
    expect(filter).toContain(ESCAPED.toLowerCase());
  });
});

describe("buildPlanFilter release and status filters", () => {
  test("adds has_release, release, and task_status", () => {
    expect(
      buildPlanFilter({
        project: "projects/p",
        hasRelease: true,
        release: "projects/p/releases/r1",
        taskStatuses: ["FAILED", "PENDING"],
      })
    ).toBe(
      'has_release == true && release == "projects/p/releases/r1" && task_status in ["FAILED", "PENDING"]'
    );
  });

  test("keeps has_release == false and skips an empty status list", () => {
    expect(
      buildPlanFilter({
        project: "projects/p",
        hasRelease: false,
        taskStatuses: [],
      })
    ).toBe("has_release == false");
  });
});

describe("buildPlanFindBySearchParams", () => {
  test("maps status scopes to task statuses", () => {
    const find = buildPlanFindBySearchParams({
      query: "",
      scopes: [
        { id: "project", value: "p" },
        { id: "status", value: "WAITING" },
        { id: "status", value: "FAILED" },
        { id: "status", value: "UNKNOWN" },
      ],
    });
    expect(find.taskStatuses).toEqual(["NOT_STARTED", "PENDING", "FAILED"]);
  });

  test("counts skipped tasks as done", () => {
    const find = buildPlanFindBySearchParams({
      query: "",
      scopes: [
        { id: "project", value: "p" },
        { id: "status", value: "DONE" },
      ],
    });
    expect(find.taskStatuses).toEqual(["DONE", "SKIPPED"]);
  });

  test("reads the release and created scopes", () => {
    const find = buildPlanFindBySearchParams(
      {
        query: "",
        scopes: [
          { id: "project", value: "p" },
          { id: "release", value: "projects/p/releases/r1" },
          { id: "created", value: "1000,2000" },
        ],
      },
      { hasRelease: true }
    );
    expect(find).toMatchObject({
      project: "projects/p",
      hasRelease: true,
      release: "projects/p/releases/r1",
      createdTsAfter: 1000,
      createdTsBefore: 2000,
    });
    expect(find.taskStatuses).toBeUndefined();
  });
});
