import { act, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import type { DatabaseResource } from "@/types";
import { MemberDatabaseResourceName } from "./MemberDatabaseResourceName";

const mocks = vi.hoisted(() => ({
  databasesByName: {} as Record<string, unknown>,
  instancesByName: {} as Record<string, unknown>,
}));

vi.mock("@/stores/app", () => ({
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

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: ({ engine }: { engine: string }) => (
    <span data-testid="engine-icon">{engine}</span>
  ),
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => {
    const match = name.match(
      /^(?<instance>(?:projects\/[^/]+\/)?instances\/(?<instanceName>[^/]+))\/databases\/(?<databaseName>[^/]+)$/
    );
    return {
      instance: match?.groups?.instance ?? "",
      databaseName: match?.groups?.databaseName ?? name,
      instanceName: match?.groups?.instanceName ?? "",
    };
  },
  extractInstanceResourceName: (name: string) =>
    name.match(/(?:^|\/)instances\/([^/]+)/)?.[1] ?? name,
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

  describe("a name cut by its column", () => {
    // jsdom lays nothing out, so each test says which names overflow.
    let cut: string[] = [];
    beforeEach(() => {
      cut = [];
      vi.spyOn(HTMLElement.prototype, "scrollWidth", "get").mockImplementation(
        function (this: HTMLElement) {
          return cut.includes(this.textContent ?? "") ? 200 : 100;
        }
      );
      vi.spyOn(HTMLElement.prototype, "clientWidth", "get").mockReturnValue(
        100
      );
    });
    afterEach(() => {
      vi.restoreAllMocks();
    });

    const focusName = async (name: string) => {
      await act(async () => {
        screen
          .getByText(name)
          .dispatchEvent(new FocusEvent("focusin", { bubbles: true }));
      });
    };
    const overlayText = () =>
      document.getElementById("bb-react-layer-overlay")?.textContent ?? "";

    test("shows the full instance title on hover", async () => {
      cut = ["Production"];
      render(<MemberDatabaseResourceName resource={resource} />);

      await focusName("Production");
      expect(overlayText()).toBe("Production");
    });

    test("shows the full database name on hover", async () => {
      cut = ["hr"];
      render(<MemberDatabaseResourceName resource={resource} />);

      await focusName("hr");
      expect(overlayText()).toBe("hr");
    });

    test("shows no tooltip for a name that fits", async () => {
      render(<MemberDatabaseResourceName resource={resource} />);

      await focusName("hr");
      expect(overlayText()).toBe("");
    });
  });

  test("renders wildcard for unscoped resources", () => {
    render(<MemberDatabaseResourceName />);

    expect(screen.getByText("*")).toBeInTheDocument();
  });

  test("looks up a project instance by its canonical parent", () => {
    const projectResource: DatabaseResource = {
      databaseFullName: "projects/app/instances/prod/databases/hr",
    };
    mocks.databasesByName = {
      [projectResource.databaseFullName]: {
        name: projectResource.databaseFullName,
      },
    };
    mocks.instancesByName = {
      "projects/app/instances/prod": {
        name: "projects/app/instances/prod",
        title: "Project production",
        engine: "POSTGRES",
      },
    };

    render(<MemberDatabaseResourceName resource={projectResource} />);

    expect(screen.getByText("Project production")).toBeInTheDocument();
    expect(screen.getByText("hr")).toBeInTheDocument();
  });
});
