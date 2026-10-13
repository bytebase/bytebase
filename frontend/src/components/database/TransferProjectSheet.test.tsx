import { render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { describe, expect, test, vi } from "vitest";

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/database", () => ({
  DatabaseTableView: () => null,
}));
vi.mock("@/components/ProjectSelect", () => ({
  ProjectSelect: ({ value }: { value: string }) => (
    <div data-testid="project-select" data-value={value} />
  ),
}));
vi.mock("@/components/ui/radio-group", () => ({
  RadioGroup: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  RadioGroupItem: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/components/ui/sheet", () => ({
  Sheet: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetContent: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetHeader: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetTitle: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetBody: ({ children }: { children: ReactNode }) => <div>{children}</div>,
  SheetFooter: ({ children }: { children: ReactNode }) => <div>{children}</div>,
}));
vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: { serverInfo: { defaultProject: string } }) => unknown) =>
    selector({ serverInfo: { defaultProject: "projects/default" } }),
}));

import { TransferProjectSheet } from "./TransferProjectSheet";

describe("TransferProjectSheet", () => {
  test("prefills the destination supplied by Project Home", () => {
    render(
      <TransferProjectSheet
        open
        databases={[]}
        initialProjectName="projects/orders"
        onClose={() => {}}
        onTransfer={vi.fn()}
      />
    );

    expect(screen.getByTestId("project-select").getAttribute("data-value"))
      .toBe("projects/orders");
  });
});
