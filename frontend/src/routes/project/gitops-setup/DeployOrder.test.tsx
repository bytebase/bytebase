import { create } from "@bufbuild/protobuf";
import { act, type ReactNode } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Database,
  DatabaseSchema$,
} from "@/types/proto-es/v1/database_service_pb";
import { DeployOrder } from "./DeployOrder";
import type { DeployTargets } from "./useDeployTargets";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({
    t: (key: string, options?: Record<string, unknown>) =>
      options ? `${key} ${JSON.stringify(options)}` : key,
  }),
}));

vi.mock("@/components/DatabaseTargetDisplay", () => ({
  DatabaseTargetDisplay: ({ database }: { database: Database }) => (
    <span data-testid="database">{database.name}</span>
  ),
}));

vi.mock("@/components/EnvironmentLabel", () => ({
  EnvironmentLabel: ({ environmentName }: { environmentName: string }) => (
    <span data-testid="environment">{environmentName}</span>
  ),
}));

vi.mock("@/components/ui/popover", () => ({
  Popover: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  PopoverTrigger: ({ children }: { children: ReactNode }) => (
    <button type="button" data-testid="more">
      {children}
    </button>
  ),
  PopoverContent: ({ children }: { children: ReactNode }) => (
    <div data-testid="more-list">{children}</div>
  ),
}));

const db = (name: string) =>
  create(DatabaseSchema$, { name: `instances/pg/databases/${name}` });

const many = (prefix: string, count: number) =>
  Array.from({ length: count }, (_, i) => db(`${prefix}${i + 1}`));

describe("DeployOrder", () => {
  let container: HTMLDivElement;
  let root: ReturnType<typeof createRoot>;

  const render = (targets: DeployTargets, unsupported: Database[] = []) => {
    act(() =>
      root.render(<DeployOrder targets={targets} unsupported={unsupported} />)
    );
  };
  const openList = () => {
    const trigger = [...container.querySelectorAll("button")].find((b) =>
      b.textContent?.includes("gitops.setup.deploy-order.label")
    );
    act(() => trigger?.click());
  };

  beforeEach(() => {
    container = document.createElement("div");
    document.body.appendChild(container);
    root = createRoot(container);
  });

  afterEach(() => {
    act(() => root.unmount());
    container.remove();
  });

  test("sums up the environments in order, with the list behind a toggle", () => {
    render({
      stages: [
        {
          environment: "environments/test",
          id: "test",
          databases: many("t", 4),
        },
        { environment: "environments/prod", id: "prod", databases: [db("p1")] },
      ],
      unassigned: [],
      total: 5,
      loading: false,
    });
    expect(container.textContent).toContain(
      'database-count {\\"count\\":5}'
    );
    const environments = () =>
      [...container.querySelectorAll('[data-testid="environment"]')].map(
        (e) => e.textContent
      );
    expect(environments()).toEqual(["environments/test", "environments/prod"]);
    expect(container.textContent).toContain("(4)");
    expect(container.querySelector("li")).toBeNull();

    openList();
    expect(container.querySelectorAll("li")).toHaveLength(2);
  });

  test("lists the databases the workflow type does not support", () => {
    const stage = {
      environment: "environments/prod",
      id: "prod",
      databases: [db("p1"), db("mongo1")],
    };
    const targets = { stages: [stage], unassigned: [], total: 2, loading: false };
    render(targets);
    expect(container.textContent).not.toContain("unsupported-title");

    render(targets, [db("mongo1")]);
    expect(container.textContent).toContain(
      'gitops.setup.deploy-order.unsupported-title {"count":1}'
    );
    expect(
      [...container.querySelectorAll('[data-testid="database"]')].map(
        (e) => e.textContent
      )
    ).toEqual(["instances/pg/databases/mongo1"]);
  });

  test("the list shows three chips, then +N more", () => {
    render({
      stages: [
        {
          environment: "environments/prod",
          id: "prod",
          databases: many("p", 5),
        },
      ],
      unassigned: [],
      total: 5,
      loading: false,
    });
    openList();
    const section = container.querySelector("li");
    const chips = [
      ...(section?.querySelectorAll('[data-testid="database"]') ?? []),
    ].filter((e) => !e.closest('[data-testid="more-list"]'));
    expect(chips.map((e) => e.textContent)).toEqual([
      "instances/pg/databases/p1",
      "instances/pg/databases/p2",
      "instances/pg/databases/p3",
    ]);
    expect(container.querySelector('[data-testid="more"]')?.textContent).toBe(
      'common.n-more {"n":2}'
    );
  });

  test("warns about databases with no environment, outside the toggle", () => {
    render({
      stages: [
        { environment: "environments/prod", id: "prod", databases: [db("p1")] },
      ],
      unassigned: [db("old1"), db("old2")],
      total: 3,
      loading: false,
    });
    const alert = container.querySelector('[role="alert"]');
    expect(alert?.textContent).toContain(
      'gitops.setup.deploy-order.unassigned-title {"count":2}'
    );
    expect(alert?.textContent).toContain("instances/pg/databases/old2");
    // The summary counts only the databases that are deployed.
    expect(container.textContent).toContain(
      'database-count {\\"count\\":1}'
    );
  });
});
