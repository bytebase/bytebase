import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Engine } from "@/types/proto-es/v1/common_pb";
import { PlanTargetDisplay } from "./PlanTargetDisplay";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

const mocks = vi.hoisted(() => ({
  databasesByName: {} as Record<string, unknown>,
  environmentList: [] as unknown[],
  getEnvironmentByName: vi.fn(),
}));

vi.mock("@/react/components/EngineIcon", () => ({
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

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/react/stores/app", () => {
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
  extractDatabaseResourceName: (name: string) => ({
    databaseName: name.split("/databases/")[1] ?? name,
  }),
  getInstanceResource: (database: {
    instanceResource?: { engine: Engine; title: string };
  }) => database.instanceResource,
}));

describe("PlanTargetDisplay", () => {
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
        <PlanTargetDisplay
          showEnvironment
          target="projects/p/instances/prod/databases/app"
        />
      );
    });

    expect(container.textContent).toContain("POSTGRES");
    expect(container.textContent).toContain("Production");
    expect(container.textContent).toContain("prod-instance");
    expect(container.textContent).toContain("app");
  });

  it("applies stable truncation priority across environment, instance, and database name", () => {
    act(() => {
      root.render(
        <PlanTargetDisplay
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
    expect(rootElement?.getAttribute("title")).toBe(
      "Production / prod-instance / app"
    );
    expect(environment?.className).toContain("max-w-24");
    expect(instance?.className).toContain("max-w-40");
    expect(database?.className).toContain("flex-1");
    expect(database?.className).toContain("min-w-12");
  });

  it("can hide optional database metadata", () => {
    act(() => {
      root.render(
        <PlanTargetDisplay
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
        <PlanTargetDisplay
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
    act(() => {
      root.render(
        <PlanTargetDisplay target="projects/p/databaseGroups/prod" />
      );
    });

    expect(container.textContent).toContain("projects/p/databaseGroups/prod");
  });
});
