import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import {
  type ButtonHTMLAttributes,
  type ReactElement,
  type ReactNode,
} from "react";
import { beforeEach, expect, test, vi } from "vitest";
import { IssueStatus } from "@/types/proto-es/v1/issue_service_pb";
import type { PlanDetailPageState } from "../shell/hooks/types";
import { PlanDetailMeta } from "./PlanDetailMeta";

const mocks = vi.hoisted(() => ({
  creationIssueLabels: [] as string[],
  page: undefined as unknown as PlanDetailPageState,
  patchState: vi.fn(),
  permissions: new Set<string>(),
  pushNotification: vi.fn(),
  setCreationIssueLabels: vi.fn(),
  updateIssue: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));

vi.mock("@/connect", () => ({
  issueServiceClientConnect: { updateIssue: mocks.updateIssue },
}));

vi.mock("@/react/components/HumanizeTs", () => ({ HumanizeTs: () => null }));

vi.mock("@/react/components/ui/checkbox", () => ({
  Checkbox: ({ checked }: { checked: boolean }) => (
    <input checked={checked} readOnly type="checkbox" />
  ),
}));

vi.mock("@/react/components/ui/popover", async () => {
  const React = await vi.importActual<typeof import("react")>("react");
  const PopoverContext = React.createContext<{
    onOpenChange?: (open: boolean) => void;
    open: boolean;
  }>({ open: false });

  return {
    Popover: ({
      children,
      onOpenChange,
      open = false,
    }: {
      children: ReactNode;
      onOpenChange?: (open: boolean) => void;
      open?: boolean;
    }) => (
      <PopoverContext.Provider value={{ onOpenChange, open }}>
        <div>{children}</div>
      </PopoverContext.Provider>
    ),
    PopoverContent: ({ children }: { children: ReactNode }) => {
      const { open } = React.useContext(PopoverContext);
      return open ? <div>{children}</div> : null;
    },
    PopoverTrigger: ({
      children,
      render,
    }: {
      children: ReactNode;
      render?: ReactElement<ButtonHTMLAttributes<HTMLButtonElement>>;
    }) => {
      const { onOpenChange, open } = React.useContext(PopoverContext);
      if (!React.isValidElement(render)) return <>{children}</>;
      const originalOnClick = render.props.onClick;
      return React.cloneElement(
        render,
        {
          onClick: (event) => {
            originalOnClick?.(event);
            onOpenChange?.(!open);
          },
        },
        children
      );
    },
  };
});

vi.mock("@/react/lib/utils", () => ({
  cn: (...classes: Array<string | false | null | undefined>) =>
    classes.filter(Boolean).join(" "),
}));

vi.mock("@/store", () => ({
  extractUserEmail: (name: string) => name,
  pushNotification: mocks.pushNotification,
}));

vi.mock("@/types", () => ({ getTimeForPbTimestampProtoEs: () => 0 }));

vi.mock("@/utils", () => ({
  colorToHex: () => "#000000",
  hasProjectPermissionV2: (_project: unknown, permission: string) =>
    mocks.permissions.has(permission),
}));

vi.mock("../shell/PlanDetailContext", () => ({
  usePlanDetailContext: () => mocks.page,
}));

const makePage = (): PlanDetailPageState =>
  ({
    creationIssueLabels: mocks.creationIssueLabels,
    isCreating: false,
    issue: {
      draft: true,
      labels: ["alpha"],
      name: "projects/p1/issues/456",
      status: IssueStatus.OPEN,
    },
    patchState: mocks.patchState,
    plan: {
      creator: "users/owner@example.com",
      name: "projects/p1/plans/123",
    },
    project: {
      issueLabels: [{ value: "alpha" }, { value: "beta" }],
      name: "projects/p1",
    },
    setCreationIssueLabels: mocks.setCreationIssueLabels,
  }) as unknown as PlanDetailPageState;

beforeEach(() => {
  vi.clearAllMocks();
  mocks.creationIssueLabels = [];
  mocks.page = makePage();
  mocks.permissions = new Set();
  mocks.updateIssue.mockImplementation(async (request) => request.issue);
});

test("edits Issue labels directly on the preview creation page", () => {
  mocks.creationIssueLabels = ["alpha"];
  mocks.page = {
    ...makePage(),
    isCreating: true,
    issue: undefined,
  };

  render(<PlanDetailMeta />);

  expect(screen.getByRole("button", { name: /alpha/ })).toBeVisible();
  fireEvent.click(screen.getByRole("button", { name: /alpha/ }));
  fireEvent.click(screen.getByRole("button", { name: "beta" }));

  expect(mocks.setCreationIssueLabels).toHaveBeenCalledWith([
    "alpha",
    "beta",
  ]);
});

test("keeps draft labels read-only without issues.update and editable with it", async () => {
  const { rerender } = render(<PlanDetailMeta />);

  expect(screen.getByRole("button", { name: "issue.labels" })).toBeDisabled();
  expect(
    screen.queryByRole("button", { name: "common.remove" })
  ).not.toBeInTheDocument();
  expect(
    screen.queryByRole("button", { name: "beta" })
  ).not.toBeInTheDocument();

  mocks.permissions = new Set(["bb.issues.update"]);
  mocks.page = {
    ...mocks.page,
    project: { ...mocks.page.project },
  };
  rerender(<PlanDetailMeta />);

  const labelsButton = screen.getByRole("button", { name: "issue.labels" });
  expect(labelsButton).toBeEnabled();
  expect(screen.getByRole("button", { name: "common.remove" })).toBeEnabled();
  fireEvent.click(labelsButton);
  fireEvent.click(screen.getByRole("button", { name: "beta" }));

  await waitFor(() => expect(mocks.updateIssue).toHaveBeenCalledOnce());
  expect(mocks.updateIssue).toHaveBeenCalledWith(
    expect.objectContaining({
      issue: expect.objectContaining({
        draft: true,
        labels: ["alpha", "beta"],
      }),
      updateMask: { paths: ["labels"] },
    })
  );
});
