import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type SharePointSetting,
  SharePointSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";

const mocks = vi.hoisted(() => ({
  stored: undefined as SharePointSetting | undefined,
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));
vi.mock("@/hooks/useAppState", () => ({ usePlanFeature: () => true }));
vi.mock("@/utils", () => ({ hasWorkspacePermissionV2: () => true }));
vi.mock("./SharePointSheet", () => ({
  SharePointSheet: ({ open }: { open: boolean }) =>
    open ? <div>sharepoint-sheet</div> : null,
}));
vi.mock("@/stores/app", () => {
  const state = () => ({
    settingsByName: {},
    getSettingByName: () =>
      mocks.stored && {
        value: {
          value: { case: "export", value: { sharepoint: mocks.stored } },
        },
      },
    getOrFetchSettingByName: async () => undefined,
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

import { ExportDestinationsPage } from "./ExportDestinationsPage";

describe("ExportDestinationsPage", () => {
  beforeEach(() => {
    mocks.stored = undefined;
  });

  test("links to the SharePoint guide", () => {
    render(<ExportDestinationsPage />);
    expect(
      screen.getByRole("link", { name: "common.learn-more" })
    ).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/security/database-permission/export/sharepoint?source=console"
    );
  });

  test("offers to configure SharePoint when it is not set up", async () => {
    render(<ExportDestinationsPage />);
    expect(
      await screen.findByText("settings.export-destinations.not-configured")
    ).toBeInTheDocument();
    fireEvent.click(
      screen.getByRole("button", {
        name: "common.configure export-data.destination.sharepoint",
      })
    );
    expect(screen.getByText("sharepoint-sheet")).toBeInTheDocument();
  });

  test("shows a configured SharePoint application", async () => {
    mocks.stored = create(SharePointSettingSchema, {
      tenantId: "tenant",
      clientId: "client",
      clientSecretSet: true,
    });
    render(<ExportDestinationsPage />);
    expect(
      await screen.findByText("settings.export-destinations.configured")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", {
        name: "common.edit export-data.destination.sharepoint",
      })
    ).toBeInTheDocument();
  });
});
