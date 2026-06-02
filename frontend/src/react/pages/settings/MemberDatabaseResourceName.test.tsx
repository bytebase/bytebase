import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { DatabaseResource } from "@/types";
import { MemberDatabaseResourceName } from "./MemberDatabaseResourceName";

const mocks = vi.hoisted(() => ({
  databasesByName: {} as Record<string, unknown>,
  instancesByName: {} as Record<string, unknown>,
}));

vi.mock("@/react/stores/app", () => ({
  useAppStore: <T,>(
    selector: (state: {
      databasesByName: Record<string, unknown>;
      instancesByName: Record<string, unknown>;
    }) => T
  ) =>
    selector({
      databasesByName: mocks.databasesByName,
      instancesByName: mocks.instancesByName,
    }),
}));

vi.mock("@/react/components/EngineIcon", () => ({
  EngineIcon: ({ engine }: { engine: string }) => (
    <span data-testid="engine-icon">{engine}</span>
  ),
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => {
    const parts = name.split("/");
    return {
      databaseName: parts[3] ?? name,
      instanceName: parts[1] ?? "",
    };
  },
  extractInstanceResourceName: (name: string) => name.split("/")[1] ?? name,
  getInstanceResource: (database: { instanceResource?: unknown }) =>
    database.instanceResource,
}));

const resource: DatabaseResource = {
  databaseFullName: "instances/prod/databases/hr",
};

describe("MemberDatabaseResourceName", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.databasesByName = {
      "instances/prod/databases/hr": {
        name: "instances/prod/databases/hr",
        instanceResource: {
          name: "instances/prod",
          title: "Production",
          engine: "POSTGRES",
        },
      },
    };
    mocks.instancesByName = {
      "instances/prod": {
        name: "instances/prod",
        title: "Production",
        engine: "POSTGRES",
      },
    };
  });

  test("renders instance title and database name instead of raw resource path", () => {
    render(<MemberDatabaseResourceName resource={resource} />);

    expect(screen.getByTestId("engine-icon")).toHaveTextContent("POSTGRES");
    expect(screen.getByText("Production")).toBeInTheDocument();
    expect(screen.getByText("hr")).toBeInTheDocument();
    expect(
      screen.queryByText(resource.databaseFullName)
    ).not.toBeInTheDocument();
  });

  test("falls back to instance id without showing unknown instance engine", () => {
    mocks.databasesByName = {
      "instances/prod/databases/hr": {
        name: "instances/prod/databases/hr",
      },
    };
    mocks.instancesByName = {
      "instances/prod": {
        name: "instances/-",
        title: "",
        engine: "MYSQL",
      },
    };

    render(<MemberDatabaseResourceName resource={resource} />);

    expect(screen.queryByTestId("engine-icon")).not.toBeInTheDocument();
    expect(screen.getByText("prod")).toBeInTheDocument();
    expect(screen.getByText("hr")).toBeInTheDocument();
  });

  test("renders wildcard for unscoped resources", () => {
    render(<MemberDatabaseResourceName />);

    expect(screen.getByText("*")).toBeInTheDocument();
  });
});
