import { describe, expect, it } from "vitest";
import { statementAsTyped } from "./statementText";

describe("statementAsTyped", () => {
  it("returns a statement no rewrite touched", () => {
    expect(statementAsTyped("SELECT * FROM t")).toBe("SELECT * FROM t");
  });

  it("drops a LIMIT the parser fitted on", () => {
    expect(statementAsTyped("SELECT * FROM t LIMIT 1000;")).toBe(
      "SELECT * FROM t"
    );
  });

  it("unwraps the single-line wrapper", () => {
    expect(
      statementAsTyped(
        "WITH result AS (SELECT * FROM t) SELECT * FROM result LIMIT 1000;"
      )
    ).toBe("SELECT * FROM t");
  });

  it("unwraps the wrapper the Postgres parser writes over several lines", () => {
    expect(
      statementAsTyped(
        "WITH result AS (\nSELECT * FROM t\n) SELECT * FROM result LIMIT 1000;"
      )
    ).toBe("SELECT * FROM t");
  });

  it("unwraps SQL Server's TOP form", () => {
    expect(
      statementAsTyped(
        "WITH result AS (SELECT * FROM t) SELECT TOP 1000 * FROM result;"
      )
    ).toBe("SELECT * FROM t");
  });

  it("unwraps BigQuery's unlimited form", () => {
    expect(
      statementAsTyped("WITH result AS (SELECT * FROM t) SELECT * FROM result;")
    ).toBe("SELECT * FROM t");
  });

  it("keeps a LIMIT the user typed inside the wrapper", () => {
    expect(
      statementAsTyped(
        "WITH result AS (SELECT * FROM t LIMIT 5) SELECT * FROM result LIMIT 1000;"
      )
    ).toBe("SELECT * FROM t LIMIT 5");
  });

  it("unwraps a statement with a CTE of its own", () => {
    expect(
      statementAsTyped(
        "WITH result AS (WITH a AS (SELECT 1 AS x) SELECT x FROM a) SELECT * FROM result LIMIT 1000;"
      )
    ).toBe("WITH a AS (SELECT 1 AS x) SELECT x FROM a");
  });

  it("leaves a statement that only looks like the wrapper alone", () => {
    expect(
      statementAsTyped("WITH result AS (SELECT 1 AS x) SELECT x FROM result")
    ).toBe("WITH result AS (SELECT 1 AS x) SELECT x FROM result");
  });
});
