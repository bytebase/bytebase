import { describe, expect, test } from "vitest";
import { skill } from "./databaseChange";

describe("database-change skill", () => {
  test("keeps DDL generation separate from persisting a Plan", () => {
    expect(skill.content).toContain(
      "schema change request on a selected @Database"
    );
    expect(skill.content).toContain("DatabaseService/GetDatabaseSchema");
    expect(skill.content).toContain("`draft-plan` action");
    expect(skill.content).toContain(
      "Do not call CreateSheet, CreatePlan, or CreateIssue"
    );
  });

  test("finds project databases through SearchDatabases", () => {
    expect(skill.content).toContain("SearchDatabases");
    expect(skill.content).toContain('project == "projects/{project-id}"');
    expect(skill.content).not.toContain("ListDatabases");
  });

  test("passes plain SQL as sheet content for call_api to encode once", () => {
    const sheetContent = /"content": "([^"]+)"/.exec(skill.content)?.[1];

    expect(sheetContent).toBe("CREATE TABLE users (id INT PRIMARY KEY);");
    expect(skill.content).toContain("Pass plain SQL as Sheet.content");
  });

  test("waits for approval rather than creating a rollout after issue creation", () => {
    expect(skill.content).toContain(
      "Do not create a rollout immediately after creating an issue"
    );
    expect(skill.content).toContain(
      '"parent": "projects/{project-id}/plans/{plan-id}"'
    );
    expect(skill.content).not.toContain('"rollout": {');
    expect(skill.content).toContain("target eligibility");
  });
});
