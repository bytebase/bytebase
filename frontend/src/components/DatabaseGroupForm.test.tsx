import type { ReactElement } from "react";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { beforeEach, describe, expect, test, vi } from "vitest";
import type { Project } from "@/types/proto-es/v1/project_service_pb";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/ExprEditor", () => ({
  ExprEditor: () => <div data-testid="expr-editor" />,
}));

vi.mock("@/components/MatchedDatabaseView", () => ({
  MatchedDatabaseView: () => <div data-testid="matched-database-view" />,
}));

vi.mock("@/components/ResourceIdField", () => ({
  ResourceIdField: () => <div data-testid="resource-id-field" />,
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: {
    getState: () => ({
      createDatabaseGroup: vi.fn(),
      getOrFetchDBGroupByName: vi.fn(),
      updateDatabaseGroup: vi.fn(),
    }),
  },
}));

let DatabaseGroupForm: typeof import("./DatabaseGroupForm").DatabaseGroupForm;

const renderIntoContainer = (element: ReactElement) => {
  const container = document.createElement("div");
  const root = createRoot(container);

  return {
    container,
    render: () =>
      act(() => {
        root.render(element);
      }),
    unmount: () =>
      act(() => {
        root.unmount();
      }),
  };
};

beforeEach(async () => {
  ({ DatabaseGroupForm } = await import("./DatabaseGroupForm"));
});

describe("DatabaseGroupForm", () => {
  test("keeps condition editor and database preview close on wide screens", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupForm
        readonly={false}
        project={{ name: "projects/sample" } as Project}
        onDismiss={vi.fn()}
      />
    );

    render();

    const exprEditor = container.querySelector('[data-testid="expr-editor"]');
    const databasePreview = container.querySelector(
      '[data-testid="matched-database-view"]'
    );
    expect(exprEditor).toBeTruthy();
    expect(databasePreview).toBeTruthy();

    const conditionColumn = exprEditor?.parentElement;
    const databaseColumn = databasePreview?.parentElement;
    const layout = conditionColumn?.parentElement;
    expect(layout?.className).toContain("xl:flex-row");
    expect(layout?.className).toContain("xl:gap-x-8");
    expect(conditionColumn?.className).toContain("xl:w-fit");
    expect(databaseColumn?.className).toContain("xl:w-[28rem]");

    unmount();
  });

  test("aligns form content and sticky footer to its parent", () => {
    const { container, render, unmount } = renderIntoContainer(
      <DatabaseGroupForm
        readonly={false}
        project={{ name: "projects/sample" } as Project}
        onDismiss={vi.fn()}
      />
    );

    render();

    const exprEditor = container.querySelector('[data-testid="expr-editor"]');
    const content = exprEditor?.closest(".flex-1.mb-6");
    expect(content?.className).not.toContain("px-4");

    const footerContent = container.querySelector(
      '[data-slot="sticky-action-footer-content"]'
    );
    expect(footerContent?.className).not.toContain("px-4");
    expect(footerContent?.className).not.toContain("sm:px-6");

    unmount();
  });
});
