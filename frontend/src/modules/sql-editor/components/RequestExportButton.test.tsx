import { render, screen } from "@testing-library/react";
import { describe, expect, test, vi } from "vitest";
import { RequestExportButton } from "./RequestExportButton";

const mocks = vi.hoisted(() => ({
  loadSubscription: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: () => null,
}));

vi.mock("@/components/PermissionGuard", () => ({
  PermissionGuard: ({
    children,
  }: {
    children: (props: { disabled: boolean }) => React.ReactNode;
  }) => <>{children({ disabled: false })}</>,
}));

vi.mock("@/hooks/useAppProject", () => ({
  useAppProject: () => ({
    name: "projects/proj1",
    allowJustInTimeAccess: true,
  }),
}));

vi.mock("@/modules/sql-editor/store/editor", () => ({
  useSQLEditorEditorState: (selector: (state: { project: string }) => unknown) =>
    selector({ project: "projects/proj1" }),
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (
    selector: (state: {
      loadSubscription: typeof mocks.loadSubscription;
      hasFeature: () => boolean;
    }) => unknown
  ) =>
    selector({
      loadSubscription: mocks.loadSubscription,
      hasFeature: () => true,
    }),
}));

vi.mock("./AccessGrantRequestDrawer", () => ({
  AccessGrantRequestDrawer: () => null,
}));

vi.mock("./RequestDrawerHost", () => ({
  useRequestDrawerHost: () => undefined,
}));

describe("RequestExportButton", () => {
  test("labels the export action instead of the access-grant mechanism", () => {
    render(
      <RequestExportButton
        statement="SELECT * FROM t"
        targets={["instances/i/databases/d"]}
      />
    );

    expect(screen.getByRole("button")).toHaveTextContent(
      "sql-editor.request-export"
    );
    expect(screen.getByRole("button")).not.toHaveTextContent(
      "sql-editor.request-access-grant"
    );
  });
});
