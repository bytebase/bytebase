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
});
