import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import {
  type VisualizerEngine,
  VISUALIZER_EXPLAIN_FORMATS,
} from "@/utils/queryPlanEngines";
import { QueryPlanView } from "./QueryPlanView";

describe("QueryPlanView", () => {
  test("uses the supplied translations for parse errors", () => {
    const translate = vi.fn((key: string) => `localized:${key}`);

    render(
      <QueryPlanView
        engine={Engine.POSTGRES}
        planSource="not JSON"
        translate={translate}
      />
    );

    expect(screen.getByText("localized:error.unreadable")).toBeVisible();
    expect(screen.getByText("localized:error.postgres-invalid")).toBeVisible();
  });

  /**
   * The key family each engine's parser owns. MariaDB is read by the MySQL
   * parser and Doris by the StarRocks one, so each reports the other's
   * messages.
   */
  const ERROR_FAMILY: Record<VisualizerEngine, string> = {
    [Engine.POSTGRES]: "postgres",
    [Engine.MSSQL]: "mssql",
    [Engine.SPANNER]: "spanner",
    [Engine.MYSQL]: "mysql",
    [Engine.MARIADB]: "mysql",
    [Engine.TIDB]: "tidb",
    [Engine.ORACLE]: "oracle",
    [Engine.CLICKHOUSE]: "clickhouse",
    [Engine.BIGQUERY]: "bigquery",
    [Engine.TRINO]: "trino",
    [Engine.OCEANBASE]: "oceanbase",
    [Engine.MONGODB]: "mongodb",
    [Engine.COCKROACHDB]: "cockroachdb",
    [Engine.SNOWFLAKE]: "snowflake",
    [Engine.HIVE]: "hive",
    [Engine.REDSHIFT]: "redshift",
    [Engine.STARROCKS]: "starrocks",
    [Engine.DORIS]: "starrocks",
  };

  test("localizes what every parser says about a plan it cannot read", () => {
    const translate = vi.fn((key: string) => `localized:${key}`);
    const engines = Object.keys(VISUALIZER_EXPLAIN_FORMATS).map(
      Number
    ) as VisualizerEngine[];

    for (const engine of engines) {
      // Nothing at all, something that is not a plan, and a document of the
      // right kind holding no plan: one message each, from every parser.
      for (const planSource of ["", "not a plan", "{}"]) {
        const { unmount } = render(
          <QueryPlanView
            engine={engine}
            planSource={planSource}
            translate={translate}
          />
        );
        // A message the view has no key for would render as its own English
        // text, leaving the title as the only localized line.
        const localized = screen
          .getAllByText(/^localized:error\./)
          .map((element) => element.textContent);
        expect(localized, `${engine} on ${JSON.stringify(planSource)}`).toEqual(
          expect.arrayContaining(["localized:error.unreadable"])
        );
        expect(localized, `${engine} on ${JSON.stringify(planSource)}`).toHaveLength(2);
        const described = localized.find(
          (text) => text !== "localized:error.unreadable"
        );
        // The keys are looked up by message, so two parsers wording a
        // message identically would quietly answer for each other. Which of
        // its three messages a parser picks for a given input is its own
        // business; whose message it is, is not.
        expect(described, `${engine} on ${JSON.stringify(planSource)}`).toMatch(
          new RegExp(
            `^localized:error\\.${ERROR_FAMILY[engine]}-(empty|invalid|no-plan)$`
          )
        );
        unmount();
      }
    }
  });

  test("shows the text plan when the structured plan cannot be parsed", () => {
    render(
      <QueryPlanView
        engine={Engine.POSTGRES}
        planSource="not JSON"
        textPlanSource="Seq Scan on t"
      />
    );

    expect(screen.getByText("Seq Scan on t")).toBeVisible();
    expect(screen.queryByText("not JSON")).toBeNull();
  });
});
