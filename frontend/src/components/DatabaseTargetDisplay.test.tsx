import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import { DatabaseTargetDisplay } from "./DatabaseTargetDisplay";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  databasesByName: {} as Record<string, unknown>,
  environmentList: [] as unknown[],
  getEnvironmentByName: vi.fn(),
}));

vi.mock("@/components/EngineIcon", () => ({
  EngineIcon: ({
    className,
    engine,
  }: {
    className?: string;
    engine: Engine;
  }) => (
    <span className={className} data-testid="engine-icon">
      {Engine[engine]}
    </span>
  ),
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({
    className,
    environment,
  }: {
    className?: string;
    environment: { title: string };
  }) => (
    <span className={className} data-testid="environment-label">
      {environment.title}
    </span>
  ),
}));

vi.mock("@/components/ui/ellipsis-text", () => ({
  EllipsisText: ({
    children,
    className,
    text,
  }: {
    children?: ReactNode;
    className?: string;
    text: string;
  }) => (
    <span className={className} data-testid={`ellipsis-${text}`}>
      {children ?? text}
    </span>
  ),
}));

vi.mock("@/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/stores/app", () => {
  const getState = () => ({
    databasesByName: mocks.databasesByName,
    environmentList: mocks.environmentList,
    getEnvironmentByName: mocks.getEnvironmentByName,
  });
  const useAppStore = <T,>(
    selector: (state: ReturnType<typeof getState>) => T
  ) => selector(getState());
  useAppStore.getState = getState;
  return { useAppStore };
});

vi.mock("@/types", () => ({
  isValidDatabaseName: (name: string) => name.includes("/databases/"),
}));

vi.mock("@/utils", () => ({
  extractDatabaseResourceName: (name: string) => {
    const [, instanceAndDatabase = ""] = name.split("/instances/");
    const [instanceName = "", databaseName = name] =
      instanceAndDatabase.split("/databases/");
    return {
      databaseName,
      instanceName,
    };
  },
}));

describe("DatabaseTargetDisplay", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  beforeEach(() => {
    container = document.createElement("div");
    document.body.append(container);
    root = createRoot(container);
    mocks.databasesByName = {
      "projects/p/instances/prod/databases/app": {
        name: "projects/p/instances/prod/databases/app",
        effectiveEnvironment: "environments/prod",
        instanceResource: {
          engine: Engine.POSTGRES,
          title: "prod-instance",
        },
      },
    };
    mocks.getEnvironmentByName.mockReturnValue({
      name: "environments/prod",
      title: "Production",
    });
  });

  afterEach(() => {
    act(() => {
      root.unmount();
    });
    container.remove();
    vi.clearAllMocks();
  });

  it("renders a database target with engine, environment, instance, and database name", () => {
    act(() => {
      root.render(
        <DatabaseTargetDisplay
          showEnvironment
          target="projects/p/instances/prod/databases/app"
        />
      );
    });

    expect(container.textContent).toContain("POSTGRES");
    expect(container.textContent).toContain("Production");
    expect(
      container.querySelector('[data-testid="environment-label"]')
    ).toBeTruthy();
    expect(container.textContent).toContain("prod-instance");
    expect(container.textContent).toContain("app");
    expect(container.firstElementChild?.textContent).toBe(
      "POSTGRESprod-instance / Productionapp"
    );
  });

  it("renders a passed database even when it is not in the store cache", () => {
    mocks.databasesByName = {};

    act(() => {
      root.render(
        <DatabaseTargetDisplay
          showEnvironment
          database={{
            name: "projects/p/instances/bbdev/databases/employee",
            effectiveEnvironment: "environments/prod",
            instanceResource: {
              engine: Engine.POSTGRES,
              title: "bbdev",
            },
          } as Database}
        />
      );
    });

    expect(container.textContent).toContain("POSTGRES");
    expect(container.textContent).toContain("Production");
    expect(container.textContent).toContain("bbdev");
    expect(container.textContent).toContain("employee");
    expect(
      container.querySelector('[data-testid="ellipsis-employee"]')
    ).toBeTruthy();
  });

  it("highlights the search keyword in the database name", () => {
    act(() => {
      root.render(
        <DatabaseTargetDisplay
          database={{
            name: "projects/p/instances/bbdev/databases/bytebase",
            instanceResource: {
              engine: Engine.POSTGRES,
              title: "bbdev",
            },
          } as Database}
          keyword="byte"
        />
      );
    });

    const highlight = container.querySelector(
      '[data-testid="ellipsis-bytebase"] b'
    );
    expect(highlight?.textContent).toBe("byte");
    expect(highlight?.className).toContain("text-accent");
  });

  it("falls back to target path context while the database cache is missing", () => {
    mocks.databasesByName = {};

    act(() => {
      root.render(
        <DatabaseTargetDisplay
          showEnvironment
          target="projects/p/instances/bbdev/databases/employee"
        />
      );
    });

    expect(container.querySelector('[data-testid="engine-icon"]')).toBeNull();
    expect(container.textContent).not.toContain("MYSQL");
    expect(container.textContent).not.toContain("UNKNOWN");
    expect(container.textContent).toContain("bbdev");
    expect(container.textContent).toContain("employee");
    expect(
      container.querySelector('[data-testid="ellipsis-bbdev"]')
    ).toBeTruthy();
  });

  it("applies stable truncation priority across environment, instance, and database name", () => {
    act(() => {
      root.render(
        <DatabaseTargetDisplay
          showEnvironment
          target="projects/p/instances/prod/databases/app"
        />
      );
    });

    const rootElement = container.firstElementChild;
    const environment = Array.from(container.querySelectorAll("span")).find(
      (element) => element.textContent === "Production"
    );
    const instance = Array.from(container.querySelectorAll("span")).find(
      (element) => element.textContent === "prod-instance"
    );
    const database = Array.from(container.querySelectorAll("span")).find(
      (element) => element.textContent === "app"
    );

    expect(rootElement?.className).toContain("inline-flex");
    expect(rootElement?.className).toContain("max-w-full");
    expect(rootElement?.getAttribute("title")).toBeNull();
    expect(environment?.className).toContain("max-w-24");
    expect(instance?.className).toContain("max-w-40");
    expect(database?.className).toContain("flex-1");
    expect(database?.className).toContain("min-w-12");
  });

  it("uses overflow-aware tooltips for truncated identity names", () => {
    act(() => {
      root.render(
        <DatabaseTargetDisplay
          showEnvironment
          target="projects/p/instances/prod/databases/app"
        />
      );
    });

    expect(
      container.querySelector('[data-testid="ellipsis-prod-instance"]')
    ).toBeTruthy();
    expect(
      container.querySelector('[data-testid="ellipsis-Production"]')
    ).toBeTruthy();
    expect(
      container.querySelector('[data-testid="ellipsis-app"]')
    ).toBeTruthy();
  });

  it("can hide optional database metadata", () => {
    act(() => {
      root.render(
        <DatabaseTargetDisplay
          showEngine={false}
          showEnvironment={false}
          showInstance={false}
          target="projects/p/instances/prod/databases/app"
        />
      );
    });

    expect(container.textContent).not.toContain("POSTGRES");
    expect(container.textContent).not.toContain("Production");
    expect(container.textContent).not.toContain("prod-instance");
    expect(container.textContent).toContain("app");
  });

  it("supports a medium size for task target rows", () => {
    act(() => {
      root.render(
        <DatabaseTargetDisplay
          showEnvironment
          size="md"
          target="projects/p/instances/prod/databases/app"
        />
      );
    });

    const rootElement = container.firstElementChild;
    const engineIcon = container.querySelector('[data-testid="engine-icon"]');
    const database = Array.from(container.querySelectorAll("span")).find(
      (element) => element.textContent === "app"
    );

    expect(rootElement?.className).toContain("text-base");
    expect(engineIcon?.className).toContain("h-5");
    expect(database?.className).toContain("min-w-16");
  });

  it("falls back to the raw target when the target is not a database", () => {
    const target = "projects/p/databaseGroups/prod";

    act(() => {
      root.render(<DatabaseTargetDisplay target={target} />);
    });

    expect(container.textContent).toContain(target);
    expect(
      container.querySelector(`[data-testid="ellipsis-${target}"]`)
    ).toBeTruthy();
  });
});
