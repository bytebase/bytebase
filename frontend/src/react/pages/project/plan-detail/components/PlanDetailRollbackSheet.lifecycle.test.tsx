import { act, render, screen, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Task, TaskRun } from "@/types/proto-es/v1/rollout_service_pb";

const mocks = vi.hoisted(() => ({
  previewTaskRunRollback: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: {},
  planServiceClientConnect: {},
  rolloutServiceClientConnect: {
    previewTaskRunRollback: mocks.previewTaskRunRollback,
  },
}));

vi.mock("@/react/components/monaco", () => ({
  ReadonlyMonaco: ({ content }: { content: string }) => (
    <div data-testid="rollback-statement">{content}</div>
  ),
}));

vi.mock("@/react/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => <>{children}</>,
  usePermissionCheck: () => [true, undefined],
}));

vi.mock("@/react/components/ui/alert", () => ({
  Alert: ({ description }: { description: ReactNode }) => (
    <div role="alert">{description}</div>
  ),
}));

vi.mock("@/react/components/ui/badge", () => ({
  Badge: ({ children }: { children: ReactNode }) => <span>{children}</span>,
}));

vi.mock("@/react/components/ui/button", () => ({
  Button: ({
    children,
    ...props
  }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props}>{children}</button>
  ),
}));

vi.mock("@/react/components/ui/checkbox", () => ({
  Checkbox: () => <input type="checkbox" />,
}));

vi.mock("@/react/components/ui/sheet", () => ({
  Sheet: ({ children, open }: { children: ReactNode; open: boolean }) =>
    open ? <>{children}</> : null,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
  SheetFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <h2>{children}</h2>,
}));

vi.mock("@/react/hooks/useAppState", () => ({
  useCurrentUser: () => ({ name: "users/me@example.com" }),
}));

vi.mock("@/react/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/p1" }),
}));

vi.mock("@/react/router", () => ({ router: { push: vi.fn() } }));

vi.mock("@/react/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: unknown) => unknown) =>
      selector({ projectsByName: { "projects/p1": { name: "projects/p1" } } }),
    { getState: () => ({ createSheet: vi.fn() }) }
  ),
}));

vi.mock("@/store", () => ({ pushNotification: vi.fn() }));

vi.mock("@/utils", () => ({
  extractPlanUID: () => "1",
  extractProjectResourceName: () => "p1",
}));

vi.mock("./PlanTargetDisplay", () => ({
  PlanTargetDisplay: ({ target }: { target: string }) => <span>{target}</span>,
}));

vi.mock("./rollbackDraft", () => ({ createRollbackDraftReview: vi.fn() }));

import { PlanDetailRollbackSheet } from "./PlanDetailRollbackSheet";

type PreviewResult = { statement: string };

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((next) => {
    resolve = next;
  });
  return { promise, resolve };
}

const task = (name: string) =>
  ({ name: `tasks/${name}`, target: `instances/i/databases/${name}` }) as Task;
const taskRun = (name: string) => ({ name: `taskRuns/${name}` }) as TaskRun;
const itemsA = [{ task: task("a"), taskRun: taskRun("a") }];
const itemsB = [{ task: task("b"), taskRun: taskRun("b") }];

describe("PlanDetailRollbackSheet request lifecycle", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  test("ignores a preview response from a prior open entity", async () => {
    const previewA = deferred<PreviewResult>();
    const previewB = deferred<PreviewResult>();
    mocks.previewTaskRunRollback.mockImplementation(
      (request: { name: string }) =>
        request.name === "taskRuns/a" ? previewA.promise : previewB.promise
    );
    const onOpenChange = vi.fn();
    const { rerender } = render(
      <PlanDetailRollbackSheet
        items={itemsA}
        onOpenChange={onOpenChange}
        open
        projectName="projects/p1"
        rolloutName="projects/p1/plans/1/rollout"
      />
    );
    await waitFor(() =>
      expect(mocks.previewTaskRunRollback).toHaveBeenCalled()
    );

    rerender(
      <PlanDetailRollbackSheet
        items={itemsA}
        onOpenChange={onOpenChange}
        open={false}
        projectName="projects/p1"
        rolloutName="projects/p1/plans/1/rollout"
      />
    );
    rerender(
      <PlanDetailRollbackSheet
        items={itemsB}
        onOpenChange={onOpenChange}
        open
        projectName="projects/p1"
        rolloutName="projects/p1/plans/2/rollout"
      />
    );

    previewB.resolve({ statement: "ROLLBACK B" });
    await waitFor(() =>
      expect(screen.getByTestId("rollback-statement")).toHaveTextContent(
        "ROLLBACK B"
      )
    );

    await act(async () => {
      previewA.resolve({ statement: "ROLLBACK A" });
      await Promise.resolve();
      await Promise.resolve();
    });

    expect(screen.getByTestId("rollback-statement")).toHaveTextContent(
      "ROLLBACK B"
    );
    expect(screen.queryByText("ROLLBACK A")).toBeNull();
  });
});
