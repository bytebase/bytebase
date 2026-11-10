import type { ReactNode } from "react";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  createProject: vi.fn(),
  navigatePush: vi.fn(),
  notify: vi.fn(),
  onClose: vi.fn(),
  setRecentProject: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/app/router", () => ({
  useNavigate: () => ({ push: mocks.navigatePush }),
}));
vi.mock("@/hooks/useAppState", () => ({
  isConnectAlreadyExists: () => false,
  useCreateProject: () => ({
    createProject: mocks.createProject,
    setRecentProject: mocks.setRecentProject,
  }),
  useNotify: () => mocks.notify,
  useWorkspacePermission: () => true,
}));
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ fetchProject: vi.fn() }) },
}));
vi.mock("@/components/ResourceIdField", () => ({
  ResourceIdField: ({
    onChange,
    onValidationChange,
  }: {
    onChange: (value: string) => void;
    onValidationChange: (valid: boolean) => void;
  }) => (
    <button
      type="button"
      onClick={() => {
        onChange("new-project");
        onValidationChange(true);
      }}
    >
      set-resource-id
    </button>
  ),
}));
vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ open, children }: { open: boolean; children: ReactNode }) =>
    open ? <div>{children}</div> : null,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));

import { ProjectCreateDialog } from "./ProjectCreateDialog";

beforeEach(() => {
  vi.clearAllMocks();
  mocks.createProject.mockResolvedValue({
    name: "projects/new-project",
    title: "New project",
  });
});

describe("ProjectCreateDialog", () => {
  test("opens the new project's Home after creation", async () => {
    render(<ProjectCreateDialog open onClose={mocks.onClose} />);
    fireEvent.click(screen.getByRole("button", { name: "set-resource-id" }));
    await waitFor(() =>
      expect(screen.getByRole("button", { name: "common.create" }).hasAttribute("disabled")).toBe(false)
    );

    fireEvent.click(screen.getByRole("button", { name: "common.create" }));

    await waitFor(() =>
      expect(mocks.navigatePush).toHaveBeenCalledWith({
        name: "workspace.project.detail",
        params: { projectId: "new-project" },
      })
    );
    expect(mocks.setRecentProject).toHaveBeenCalledWith("projects/new-project");
    expect(mocks.onClose).toHaveBeenCalledOnce();
  });
});
