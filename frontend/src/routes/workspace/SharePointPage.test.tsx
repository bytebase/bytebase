import { create } from "@bufbuild/protobuf";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type SettingValue,
  type SharePointSetting,
  SharePointSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";

const mocks = vi.hoisted(() => ({
  stored: undefined as SharePointSetting | undefined,
  hasFeature: true,
  upsertSetting: vi.fn(
    async (_params: {
      value: SettingValue;
      updateMask: { paths: string[] };
    }) => undefined
  ),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));
vi.mock("@/hooks/useAppState", () => ({
  usePlanFeature: () => mocks.hasFeature,
}));
vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: () => {},
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/utils", () => ({ hasWorkspacePermissionV2: () => true }));
vi.mock("@/stores/app", () => {
  const state = () => ({
    settingsByName: {},
    getSettingByName: () =>
      mocks.stored && {
        value: { value: { case: "sharepoint", value: mocks.stored } },
      },
    getOrFetchSettingByName: async () => undefined,
    upsertSetting: mocks.upsertSetting,
  });
  return {
    useAppStore: Object.assign(
      (selector: (s: ReturnType<typeof state>) => unknown) => selector(state()),
      { getState: state }
    ),
  };
});

import { buildSharePointUpdate, SharePointPage } from "./SharePointPage";

const storedSetting = () =>
  create(SharePointSettingSchema, {
    tenantId: "tenant",
    clientId: "client",
    clientSecretSet: true,
  });

describe("buildSharePointUpdate", () => {
  test("masks only the changed fields and keeps the stored secret", () => {
    const { paths, setting } = buildSharePointUpdate(storedSetting(), {
      tenantId: " tenant-2 ",
      clientId: "client",
      clientSecret: undefined,
    });
    expect(paths).toEqual(["value.sharepoint.tenant_id"]);
    expect(setting.tenantId).toBe("tenant-2");
    expect(setting.clientSecret).toBe("");
  });

  test("masks a replacement secret", () => {
    expect(
      buildSharePointUpdate(storedSetting(), {
        tenantId: "tenant",
        clientId: "client",
        clientSecret: "new-secret",
      }).paths
    ).toEqual(["value.sharepoint.client_secret"]);
  });
});

describe("SharePointPage", () => {
  beforeEach(() => {
    mocks.stored = storedSetting();
    mocks.hasFeature = true;
    mocks.upsertSetting.mockClear();
  });

  test("shows that a secret is saved without revealing it", async () => {
    render(<SharePointPage />);
    expect(
      await screen.findByText("settings.sharepoint.client-secret-saved")
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("settings.sharepoint.client-secret")
    ).toHaveValue("");
  });

  test("saves a changed client ID", async () => {
    render(<SharePointPage />);
    fireEvent.change(
      await screen.findByLabelText("settings.sharepoint.client-id"),
      { target: { value: "client-2" } }
    );
    fireEvent.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() => expect(mocks.upsertSetting).toHaveBeenCalled());
    const params = mocks.upsertSetting.mock.calls[0][0];
    expect(params.updateMask.paths).toEqual(["value.sharepoint.client_id"]);
    expect(params.value.value).toMatchObject({
      case: "sharepoint",
      value: { clientId: "client-2" },
    });
  });

  test("requires a secret before the first save", async () => {
    mocks.stored = undefined;
    render(<SharePointPage />);
    fireEvent.change(
      await screen.findByLabelText("settings.sharepoint.tenant-id"),
      { target: { value: "tenant" } }
    );
    fireEvent.change(screen.getByLabelText("settings.sharepoint.client-id"), {
      target: { value: "client" },
    });
    expect(screen.getByRole("button", { name: "common.save" })).toBeDisabled();
  });

  test("disables editing without the plan feature", async () => {
    mocks.hasFeature = false;
    render(<SharePointPage />);
    expect(
      await screen.findByLabelText("settings.sharepoint.tenant-id")
    ).toBeDisabled();
  });
});
