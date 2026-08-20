import { create } from "@bufbuild/protobuf";
import { act } from "react";
import { createRoot } from "react-dom/client";
import { afterEach, describe, expect, test, vi } from "vitest";
import { WebhookType } from "@/types/proto-es/v1/common_pb";
import {
  Activity_Type,
  ProjectSchema,
  WebhookSchema,
} from "@/types/proto-es/v1/project_service_pb";
import { ProjectWebhookForm } from "./ProjectWebhookForm";

(globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }).IS_REACT_ACT_ENVIRONMENT =
  true;

const mocks = vi.hoisted(() => ({
  getOrFetchSettingByName: vi.fn(),
}));

vi.mock("react-i18next", async (importOriginal) => ({
  ...(await importOriginal<typeof import("react-i18next")>()),
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/app/router", () => ({
  router: {
    push: vi.fn(),
    resolve: vi.fn(() => ({ href: "#" })),
  },
}));

vi.mock("@/components/ExternalUrlAlert", () => ({
  ExternalUrlAlert: () => null,
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/types", () => ({
  projectWebhookV1TypeItemList: () => [
    {
      type: WebhookType.SLACK,
      name: "Slack",
      supportDirectMessage: false,
    },
  ],
  projectWebhookV1ActivityItemList: () => [
    {
      activity: Activity_Type.ISSUE_CREATED,
      title: "Issue creation",
      label: "When an issue is created",
      supportDirectMessage: false,
    },
  ],
}));

const store = {
  createProjectWebhook: vi.fn(),
  updateProjectWebhook: vi.fn(),
  deleteProjectWebhook: vi.fn(),
  testProjectWebhook: vi.fn(),
  settingsByName: new Map(),
  getSettingByName: vi.fn(),
  getOrFetchSettingByName: mocks.getOrFetchSettingByName,
};

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (state: typeof store) => unknown) => selector(store),
    { getState: () => store }
  ),
}));

const rendered: Array<{
  container: HTMLDivElement;
  root: ReturnType<typeof createRoot>;
}> = [];

afterEach(() => {
  for (const { container, root } of rendered.splice(0)) {
    act(() => root.unmount());
    container.remove();
  }
  vi.clearAllMocks();
});

const renderForm = (createMode: boolean) => {
  const container = document.createElement("div");
  document.body.appendChild(container);
  const root = createRoot(container);
  rendered.push({ container, root });

  act(() => {
    root.render(
      <ProjectWebhookForm
        allowEdit={true}
        create={createMode}
        project={create(ProjectSchema, { name: "projects/p" })}
        webhook={create(WebhookSchema, {
          name: createMode ? "" : "projects/p/webhooks/w",
          title: "release channel",
          type: WebhookType.SLACK,
          notificationTypes: [Activity_Type.ISSUE_CREATED],
        })}
      />
    );
  });

  return container;
};

describe("ProjectWebhookForm", () => {
  test("treats an empty URL on a saved webhook as redacted", () => {
    const container = renderForm(false);

    expect(container.textContent).toContain("project.webhook.url-hidden");
  });

  test("treats an empty URL on a new webhook as missing", () => {
    const container = renderForm(true);

    expect(container.textContent).not.toContain("project.webhook.url-hidden");
  });
});
