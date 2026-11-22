import type { ReactNode } from "react";
import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";

const workspaceID = "eJ3L1eOJf3EXzEgr";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: () => null,
}));
vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({ children }: { children: ReactNode }) => children,
  usePermissionCheck: () => [true],
}));
vi.mock("@/components/ui/button", () => ({
  Button: ({ children }: { children: ReactNode }) => <button>{children}</button>,
}));
vi.mock("@/components/ui/copy-button", () => ({
  CopyButton: ({ content }: { content: string }) => (
    <button aria-label="Copy workspace ID" data-content={content} />
  ),
}));
vi.mock("@/components/ui/form", () => ({
  FormField: ({ children, title }: { children: ReactNode; title: ReactNode }) => (
    <div>{title}{children}</div>
  ),
  FormFieldGroup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  FormSection: ({ children }: { children: ReactNode }) => <section>{children}</section>,
}));
vi.mock("@/components/ui/input", () => ({
  Input: (props: React.InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));
vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: () => true,
  useWorkspace: () => ({
    name: `workspaces/${workspaceID}`,
    title: "Workspace title",
  }),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: { updateWorkspace: () => void }) => unknown) =>
    selector({ updateWorkspace: vi.fn() }),
}));

import { BrandingSection } from "./BrandingSection";

describe("BrandingSection", () => {
  test("places a copy button beside the workspace ID", () => {
    render(<BrandingSection title="Branding" onDirtyChange={vi.fn()} />);

    expect(screen.getByDisplayValue(workspaceID)).toBeDisabled();
    expect(screen.getByRole("button", { name: "Copy workspace ID" })).toHaveAttribute(
      "data-content",
      workspaceID
    );
  });
});
