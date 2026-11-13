import { create } from "@bufbuild/protobuf";
import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  Setting_SettingName,
  type SettingValue,
  SharePointSettingSchema,
} from "@/types/proto-es/v1/setting_service_pb";

const mocks = vi.hoisted(() => ({
  upsertSetting: vi.fn(
    async (_params: {
      name: Setting_SettingName;
      value: SettingValue;
      updateMask: { paths: string[] };
    }) => undefined
  ),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/stores/app", () => ({
  useAppStore: { getState: () => ({ upsertSetting: mocks.upsertSetting }) },
}));

import { buildSharePointUpdate, SharePointSheet } from "./SharePointSheet";

const storedSetting = () =>
  create(SharePointSettingSchema, {
    tenantId: "tenant",
    clientId: "client",
    clientSecretSet: true,
  });

const renderSheet = ({
  stored = storedSetting(),
  allowEdit = true,
  onClose = vi.fn(),
} = {}) => {
  render(
    <SharePointSheet
      open
      sessionKey={1}
      stored={stored}
      allowEdit={allowEdit}
      onClose={onClose}
    />
  );
  return { onClose };
};

describe("buildSharePointUpdate", () => {
  test("masks only the changed fields and keeps the stored secret", () => {
    const { paths, setting } = buildSharePointUpdate(storedSetting(), {
      tenantId: " tenant-2 ",
      clientId: "client",
      clientSecret: undefined,
    });
    expect(paths).toEqual(["value.export.sharepoint.tenant_id"]);
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
    ).toEqual(["value.export.sharepoint.client_secret"]);
  });
});

describe("SharePointSheet", () => {
  beforeEach(() => {
    mocks.upsertSetting.mockClear();
  });

  test("links to the setup guide", () => {
    renderSheet();
    expect(
      screen.getByRole("link", { name: "common.learn-more" })
    ).toHaveAttribute(
      "href",
      "https://docs.bytebase.com/security/database-permission/export/sharepoint?source=console"
    );
  });

  test("shows that a secret is saved without revealing it", () => {
    renderSheet();
    expect(
      screen.getByText("settings.sharepoint.client-secret-saved")
    ).toBeInTheDocument();
    expect(
      screen.getByLabelText("settings.sharepoint.client-secret")
    ).toHaveValue("");
  });

  test("saves a changed client ID and closes", async () => {
    const { onClose } = renderSheet();
    fireEvent.change(screen.getByLabelText("settings.sharepoint.client-id"), {
      target: { value: "client-2" },
    });
    fireEvent.click(screen.getByRole("button", { name: "common.save" }));

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    const params = mocks.upsertSetting.mock.calls[0][0];
    expect(params.name).toBe(Setting_SettingName.EXPORT);
    expect(params.updateMask.paths).toEqual([
      "value.export.sharepoint.client_id",
    ]);
    expect(params.value.value).toMatchObject({
      case: "export",
      value: { sharepoint: { clientId: "client-2" } },
    });
  });

  test("requires a secret before the first save", () => {
    renderSheet({ stored: create(SharePointSettingSchema, {}) });
    fireEvent.change(screen.getByLabelText("settings.sharepoint.tenant-id"), {
      target: { value: "tenant" },
    });
    fireEvent.change(screen.getByLabelText("settings.sharepoint.client-id"), {
      target: { value: "client" },
    });
    expect(screen.getByRole("button", { name: "common.save" })).toBeDisabled();
    expect(
      screen.queryByRole("button", { name: "common.remove" })
    ).not.toBeInTheDocument();
  });

  test("removes the application after confirmation", async () => {
    const { onClose } = renderSheet();
    fireEvent.click(screen.getByRole("button", { name: "common.remove" }));
    const confirm = await screen.findByRole("alertdialog");
    fireEvent.click(
      within(confirm).getByRole("button", { name: "common.remove" })
    );

    await waitFor(() => expect(onClose).toHaveBeenCalled());
    const params = mocks.upsertSetting.mock.calls[0][0];
    expect(params.updateMask.paths).toEqual([
      "value.export.sharepoint.tenant_id",
      "value.export.sharepoint.client_id",
    ]);
    expect(params.value.value).toMatchObject({
      case: "export",
      value: { sharepoint: { tenantId: "", clientId: "" } },
    });
  });

  test("is read-only without edit access", () => {
    renderSheet({ allowEdit: false });
    expect(screen.getByLabelText("settings.sharepoint.tenant-id")).toBeDisabled();
    expect(
      screen.queryByRole("button", { name: "common.save" })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole("button", { name: "common.remove" })
    ).not.toBeInTheDocument();
  });
});
