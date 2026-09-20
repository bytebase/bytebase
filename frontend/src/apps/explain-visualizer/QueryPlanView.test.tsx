import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
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
