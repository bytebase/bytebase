import type {
  ButtonHTMLAttributes,
  InputHTMLAttributes,
  ReactNode,
} from "react";
import { act } from "react";
import { createRoot, type Root } from "react-dom/client";
import { afterEach, beforeEach, describe, expect, test, vi } from "vitest";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION } from "@/app/router/handles";
import { pushNotification } from "@/stores";
import { applyMaskingExemptionDraftInEditor, getMaskingExemptionEditorSnapshot, openMaskingExemptionDraftInEditor } from "@/modules/agent/logic/maskingExemptionDraftTarget";

(
  globalThis as { IS_REACT_ACT_ENVIRONMENT?: boolean }
).IS_REACT_ACT_ENVIRONMENT = true;

// ============================================================
// BYT-9788 — REGRESSION test for PR #20683.
//
// Drives the REAL create-page builder (onSubmit) and asserts that the generated
// masking-exemption CEL wraps the whole resource OR-group BEFORE the expiration
// is AND-joined, so `request.time` binds to every resource:
//   request.time < timestamp("…") && ((c1) || (c2))
//
// The pre-fix builder produced `(c1) || (c2) && request.time < …`, which cel-go
// reads as `(c1) || ((c2) && time)` — the first resource never expired.
// (#20687 later shared this builder with GrantAccessDialog, fixing BYT-9791.)
// ============================================================

type NavigationGuard = Parameters<typeof import("@/app/router").router.beforeEach>[0];

const mocks = vi.hoisted(() => ({
  identity: { userName: "users/member@example.com", workspaceName: "workspaces/default" },
  currentChatId: "chat",
  openAgent: vi.fn(),
  upsertPolicy: vi.fn(),
  getOrFetchPolicyByParentAndType: vi.fn(),
  getOrFetchSettingByName: vi.fn(),
  classification: vi.fn(() => []),
  buildCELExpr: vi.fn(),
  batchConvertParsedExprToCELString: vi.fn(),
  getExpressionsForDatabaseResource: vi.fn(
    (_resource: { table?: string }): string[] => []
  ),
  routerBack: vi.fn(),
  routerPush: vi.fn(),
  routerGuards: new Set<NavigationGuard>(),
  navigationBlocked: false,
  markDraftCreated: vi.fn(),
  parseDraftCondition: vi.fn(),
  validateDraft: vi.fn(),
  refreshSubscription: vi.fn(),
  accountSelectProps: { value: undefined as { accountParents: string[] } | undefined },
}));

vi.mock("@/modules/agent/logic/maskingExemptionDraft", () => ({
  parseMaskingExemptionDraftCondition: mocks.parseDraftCondition,
  validateMaskingExemptionDraft: mocks.validateDraft,
}));
vi.mock("@/modules/agent/store/agent", () => ({ useAgentStore: { getState: () => ({ currentChatId: mocks.currentChatId, getChat: () => ({ workspaceName: "workspaces/default", archived: false }), markMaskingExemptionDraftCreated: mocks.markDraftCreated, open: mocks.openAgent }) } }));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/modules/cel", () => ({
  ExprType: {
    Condition: "Condition",
    ConditionGroup: "ConditionGroup",
  },
  buildCELExpr: mocks.buildCELExpr,
  emptySimpleExpr: vi.fn(() => ({ type: "ConditionGroup", args: [] })),
  validateSimpleExpr: vi.fn(() => true),
  wrapAsGroup: vi.fn((expr) => expr),
}));

vi.mock("@/components/AccountMultiSelect", () => ({
  AccountMultiSelect: ({
    value,
    onChange,
    accountParents,
  }: {
    value: string[];
    onChange: (value: string[]) => void;
    accountParents: string[];
  }) => {
    mocks.accountSelectProps.value = { accountParents };
    return (
      <button
        data-testid="account-multi-select"
        onClick={() => onChange([...value, "users/alice"])}
        type="button"
      />
    );
  },
}));

vi.mock("@/components/DatabaseResourceSelector", () => ({
  DatabaseResourceSelector: ({
    onChange,
  }: {
    onChange?: (resources: unknown[]) => void;
  }) => (
    <button
      data-testid="set-two-resources"
      type="button"
      onClick={() =>
        onChange?.([
          {
            databaseFullName: "instances/inst1/databases/db1",
            schema: "public",
            table: "t1",
          },
          {
            databaseFullName: "instances/inst1/databases/db1",
            schema: "public",
            table: "t2",
          },
        ])
      }
    />
  ),
}));

vi.mock("@/components/ExprEditor", () => ({
  ExprEditor: ({ onUpdate }: { onUpdate: (expr: unknown) => void }) => (
    <button
      data-testid="set-expr"
      type="button"
      onClick={() => onUpdate({ type: "ConditionGroup", args: ["set"] })}
    />
  ),
}));

vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => <div data-testid="feature-attention" />,
}));

vi.mock("@/components/FeatureBadge", () => ({
  FeatureBadge: ({ clickable }: { clickable?: boolean }) => (
    <div data-clickable={String(clickable)} data-testid="feature-badge" />
  ),
}));

vi.mock("@/components/ui/feature-modal", () => ({
  FeatureModal: ({ open }: { open: boolean }) =>
    open ? <div data-testid="feature-modal" /> : null,
}));

vi.mock("@/components/ui/button", () => ({
  Button: (props: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props} />
  ),
}));

vi.mock("@/components/ui/expiration-picker", () => ({
  ExpirationPicker: ({ onChange }: { onChange?: (value: string) => void }) => (
    <button
      data-testid="set-expiration"
      type="button"
      onClick={() => onChange?.("2026-06-02T10:00:00.000Z")}
    />
  ),
}));

vi.mock("@/components/ui/input", () => ({
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
}));

vi.mock("@/components/ui/tooltip", () => ({
  Tooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => ({ name: "projects/proj1" }),
}));

vi.mock("@/lib/sensitive-data/components-utils", () => ({
  getClassificationLevelOptions: () => [],
}));

vi.mock("@/lib/sensitive-data/exemptionDataUtils", () => ({
  rewriteResourceDatabase: vi.fn((expression: string) => expression),
}));

vi.mock("@/lib/sensitive-data/utils", () => ({
  getExpressionsForDatabaseResource: mocks.getExpressionsForDatabaseResource,
}));

vi.mock("@/app/router", () => ({
  router: {
    back: mocks.routerBack,
    push: mocks.routerPush,
    beforeEach: (guard: NavigationGuard) => {
      mocks.routerGuards.add(guard);
      return () => mocks.routerGuards.delete(guard);
    },
  },
}));

vi.mock("@/stores", () => ({
  pushNotification: vi.fn(),
}));

vi.mock("@/stores/modules/v1/common", () => ({
  projectNamePrefix: "projects/",
}));

vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (
      selector: (state: {
        projectsByName: Record<string, unknown>;
        settingsByName: Record<string, unknown>;
        hasFeature: () => boolean;
        workspaceResourceName: () => string;
        currentUser: { name: string };
      }) => unknown
    ) =>
      selector({
        projectsByName: {},
        settingsByName: {},
        hasFeature: () => true,
        workspaceResourceName: () => mocks.identity.workspaceName,
        currentUser: { name: mocks.identity.userName },
      }),
    {
      getState: () => ({
        getOrFetchSettingByName: mocks.getOrFetchSettingByName,
        getOrFetchPolicyByParentAndType: mocks.getOrFetchPolicyByParentAndType,
        upsertPolicy: mocks.upsertPolicy,
        classification: mocks.classification,
        currentUser: { name: mocks.identity.userName },
        workspaceResourceName: () => mocks.identity.workspaceName,
        refreshSubscription: mocks.refreshSubscription,
      }),
    }
  ),
}));

vi.mock("@/utils", () => ({
  batchConvertParsedExprToCELString: mocks.batchConvertParsedExprToCELString,
  extractDatabaseResourceName: vi.fn(),
  getDatabaseNameOptionConfig: vi.fn(() => ({ options: [] })),
}));
vi.mock("@/utils/v1/cel", () => ({ batchConvertParsedExprToCELString: mocks.batchConvertParsedExprToCELString }));

vi.mock("@/utils/cel-attributes", () => ({
  CEL_ATTRIBUTE_RESOURCE_DATABASE: "resource.database",
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME: "resource.database_name",
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME: "resource.table_name",
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID: "resource.instance_id",
  CEL_ATTRIBUTE_RESOURCE_SCHEMA_NAME: "resource.schema_name",
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME: "resource.column_name",
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL: "resource.classification_level",
}));

import { ProjectMaskingExemptionCreatePage } from "./ProjectMaskingExemptionCreatePage";

const TIME = 'request.time < timestamp("2026-06-02T10:00:00.000Z")';
const c1 =
  'resource.instance_id == "inst1" && resource.database_name == "db1" && resource.table_name == "t1"';
const c2 =
  'resource.instance_id == "inst1" && resource.database_name == "db1" && resource.table_name == "t2"';

const flush = async () => {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
  });
};

const click = async (element: HTMLElement) => {
  act(() => {
    element.dispatchEvent(
      new MouseEvent("click", { bubbles: true, cancelable: true })
    );
  });
  await flush();
};

const roots = new Set<Root>();
const render = () => {
  const container = document.createElement("div");
  const root = createRoot(container);
  roots.add(root);
  act(() => {
    root.render(<ProjectMaskingExemptionCreatePage projectId="proj1" />);
  });
  return {
    container,
    unmount: () => act(() => { root.unmount(); roots.delete(root); }),
  };
};

const submittedExpression = (): string | undefined => {
  const arg = mocks.upsertPolicy.mock.calls[0]?.[0] as
    | {
        policy: {
          policy: {
            value: { exemptions: { condition?: { expression: string } }[] };
          };
        };
      }
    | undefined;
  return arg?.policy.policy.value.exemptions[0]?.condition?.expression;
};

const selectResourceMode = async (
  container: HTMLElement,
  value: "ALL" | "EXPRESSION" | "SELECT"
) => {
  const index = { ALL: 0, EXPRESSION: 1, SELECT: 2 }[value];
  await click(
    Array.from(container.querySelectorAll<HTMLElement>('[role="radio"]'))[
      index
    ]!
  );
};

const clickConfirm = async (container: HTMLElement) => {
  const confirm = Array.from(
    container.querySelectorAll<HTMLButtonElement>("button")
  ).find((b) => b.textContent === "common.confirm");
  expect(confirm).toBeTruthy();
  expect(confirm?.disabled).toBe(false);
  await click(confirm!);
};

const isNavigationGuarded = () => {
  const event = new Event("beforeunload", { cancelable: true });
  window.dispatchEvent(event);
  return event.defaultPrevented;
};

describe("ProjectMaskingExemptionCreatePage builder (BYT-9788)", () => {
  beforeEach(() => {
    mocks.identity = { userName: "users/member@example.com", workspaceName: "workspaces/default" };
    mocks.currentChatId = "chat";
    mocks.openAgent.mockReset();
    mocks.upsertPolicy.mockReset();
    mocks.routerBack.mockClear();
    mocks.routerPush.mockReset();
    mocks.navigationBlocked = false;
    mocks.routerPush.mockImplementation(async () => {
      for (const guard of mocks.routerGuards) {
        guard({} as never, {} as never, result => {
          if (result === false) mocks.navigationBlocked = true;
        });
      }
    });
    vi.spyOn(window, "confirm").mockReturnValue(false);
    mocks.markDraftCreated.mockReset();
    vi.mocked(pushNotification).mockClear();
    mocks.getOrFetchPolicyByParentAndType.mockReset();
    mocks.refreshSubscription.mockResolvedValue({});
    mocks.validateDraft.mockImplementation(async (draft) => draft);
    mocks.parseDraftCondition.mockResolvedValue({ type: "ConditionGroup", args: ["draft"] });
    mocks.buildCELExpr.mockReset();
    mocks.batchConvertParsedExprToCELString.mockReset();
    mocks.getExpressionsForDatabaseResource.mockImplementation(
      (resource: { table?: string }) => [
        'resource.instance_id == "inst1"',
        'resource.database_name == "db1"',
        `resource.table_name == "${resource.table}"`,
      ]
    );
  });

  afterEach(() => {
    act(() => { for (const root of roots) root.unmount(); });
    roots.clear();
    mocks.getExpressionsForDatabaseResource.mockReset();
    vi.restoreAllMocks();
  });

  test("keeps resource scope choices visible", () => {
    const { container, unmount } = render();

    const radios = container.querySelectorAll('[role="radio"]');
    expect(radios).toHaveLength(3);
    expect(radios[0]).toHaveAttribute("aria-checked", "true");

    const badges = container.querySelectorAll('[data-testid="feature-badge"]');
    expect(badges).toHaveLength(2);
    for (const badge of badges) {
      expect(badge.getAttribute("data-clickable")).toBe("false");
    }

    unmount();
  });

  test("opens Agent beside the title without submitting or changing the exemption form", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const { container } = render();
    await click(container.querySelector<HTMLElement>('[data-testid="account-multi-select"]')!);
    await click(container.querySelector<HTMLElement>('[data-testid="set-expiration"]')!);
    const before = await getMaskingExemptionEditorSnapshot(context);
    const title = container.querySelector("h2")!;
    const help = Array.from(title.parentElement!.querySelectorAll("button")).find(button => button.textContent === "agent.masking.ask-agent-short");

    expect(help).toBeDefined();
    expect(help).toHaveAttribute("type", "button");
    await click(help!);

    expect(mocks.openAgent).toHaveBeenCalledOnce();
    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(await getMaskingExemptionEditorSnapshot(context)).toEqual(before);
  });

  test("prefills only local fields and appends the exemption only after Confirm", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: 'resource.column_name == "email"', description: "Temporary review", expirationTimestamp: "2030-01-01T00:00:00.000Z" };
    mocks.buildCELExpr.mockResolvedValue({ parsed: true });
    mocks.batchConvertParsedExprToCELString.mockResolvedValue([draft.condition]);
    const existing = { members: ["user:existing@example.com"], condition: { expression: "existing" } };
    mocks.getOrFetchPolicyByParentAndType.mockResolvedValue({ policy: { case: "maskingExemptionPolicy", value: { exemptions: [existing] } } });
    const { container, unmount } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context); });
    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(container.querySelectorAll('[role="radio"]')[1]).toHaveAttribute("aria-checked", "true");
    expect(container.querySelector<HTMLInputElement>('input[placeholder="common.description"]')?.value).toBe(draft.description);
    const snapshot = await getMaskingExemptionEditorSnapshot(context);
    expect(snapshot).toMatchObject({ members: draft.members, expirationTimestamp: draft.expirationTimestamp, chatId: "chat" });
    await clickConfirm(container);
    const exemptions = mocks.upsertPolicy.mock.calls[0]?.[0].policy.policy.value.exemptions;
    expect(exemptions).toHaveLength(2);
    expect(exemptions[0]).toMatchObject(existing);
    expect(exemptions[1]).toMatchObject({ members: draft.members, condition: { description: draft.description } });
    expect(exemptions[1].condition.expression).toContain('timestamp("2030-01-01T00:00:00.000Z")');
    unmount();
  });

  test.each([false, true])("returns to the project exemption list without a leave warning after saving (Agent draft: %s)", async agentDraft => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "All project columns", expirationTimestamp: null };
    const { container } = render();
    if (agentDraft) {
      await act(async () => { await openMaskingExemptionDraftInEditor(draft, context); });
      expect(isNavigationGuarded()).toBe(true);
    } else {
      await click(container.querySelector<HTMLElement>('[data-testid="account-multi-select"]')!);
    }

    await clickConfirm(container);
    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    expect(mocks.markDraftCreated).not.toHaveBeenCalled();
    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(mocks.routerPush).toHaveBeenCalledExactlyOnceWith({
      name: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
      params: { projectId: "proj1" },
    });
    expect(window.confirm).not.toHaveBeenCalled();
    expect(mocks.navigationBlocked).toBe(false);
    expect(isNavigationGuarded()).toBe(false);
    const confirm = Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(button => button.textContent === "common.confirm")!;
    expect(confirm.disabled).toBe(true);
    await click(confirm);
    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    expect(mocks.routerPush).toHaveBeenCalledOnce();
  });

  test("marks only the submitted action created after saving an edited Agent draft", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const action = { draftId: "exemption-1", messageId: "message-1" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    let finish!: () => void;
    mocks.upsertPolicy.mockImplementationOnce(() => new Promise<void>(resolve => { finish = resolve; }));
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, action); });
    await click(container.querySelector<HTMLElement>('[data-testid="account-multi-select"]')!);
    expect(mocks.markDraftCreated).not.toHaveBeenCalled();
    await clickConfirm(container);
    expect(mocks.markDraftCreated).not.toHaveBeenCalled();
    await act(async () => finish());
    expect(mocks.markDraftCreated).toHaveBeenCalledExactlyOnceWith("chat", "message-1", "exemption-1");
  });

  test("does not mark a failed save created and records only the successful retry", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const action = { draftId: "exemption-1", messageId: "message-1" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    mocks.upsertPolicy.mockRejectedValueOnce(new Error("Save failed"));
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, action); });
    await clickConfirm(container);
    expect(mocks.markDraftCreated).not.toHaveBeenCalled();
    await clickConfirm(container);
    expect(mocks.markDraftCreated).toHaveBeenCalledExactlyOnceWith("chat", "message-1", "exemption-1");
  });

  test("keeps the save successful when local chat status persistence fails", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    mocks.markDraftCreated.mockImplementationOnce(() => { throw new Error("Storage quota exceeded"); });
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, { draftId: "exemption-1", messageId: "message-1" }); });
    await clickConfirm(container);
    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    expect(pushNotification).toHaveBeenCalledExactlyOnceWith({ module: "bytebase", style: "SUCCESS", title: "common.created" });
    expect(isNavigationGuarded()).toBe(false);
    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(mocks.routerPush).toHaveBeenCalledExactlyOnceWith({
      name: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
      params: { projectId: "proj1" },
    });
    expect(window.confirm).not.toHaveBeenCalled();
  });

  test("retains the saved receipt and prevents duplicate saves when redirect fails", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    mocks.routerPush.mockRejectedValueOnce(new Error("Navigation failed"));
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, { draftId: "exemption-1", messageId: "message-1" }); });

    await clickConfirm(container);
    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    expect(mocks.markDraftCreated).toHaveBeenCalledExactlyOnceWith("chat", "message-1", "exemption-1");
    expect(pushNotification).toHaveBeenCalledWith({ module: "bytebase", style: "SUCCESS", title: "common.created" });
    expect(isNavigationGuarded()).toBe(false);
    const confirm = Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(button => button.textContent === "common.confirm")!;
    expect(confirm.disabled).toBe(true);
    await click(confirm);
    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    expect(mocks.routerPush).toHaveBeenCalledOnce();
    expect(window.confirm).not.toHaveBeenCalled();
  });

  test("attributes a refined exemption to the newer action rather than the opened proposal", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, { draftId: "exemption-1", messageId: "message-1" }); });
    const snapshot = await getMaskingExemptionEditorSnapshot(context);
    await act(async () => { await applyMaskingExemptionDraftInEditor({ ...draft, description: "Updated review access" }, snapshot!, context, { draftId: "exemption-2", messageId: "message-2" }); });
    await clickConfirm(container);
    expect(mocks.markDraftCreated).toHaveBeenCalledExactlyOnceWith("chat", "message-2", "exemption-2");
  });

  test("records the submitted action even if another chat is selected during saving", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    let finish!: () => void;
    mocks.upsertPolicy.mockImplementationOnce(() => new Promise<void>(resolve => { finish = resolve; }));
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, { draftId: "exemption-1", messageId: "message-1" }); });
    await clickConfirm(container);
    mocks.currentChatId = "other-chat";
    await act(async () => finish());
    expect(mocks.markDraftCreated).toHaveBeenCalledExactlyOnceWith("chat", "message-1", "exemption-1");
  });

  test.each(["user", "workspace"])("does not update a receipt after the active %s changes during saving", async changed => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    let finish!: () => void;
    mocks.upsertPolicy.mockImplementationOnce(() => new Promise<void>(resolve => { finish = resolve; }));
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, { draftId: "exemption-1", messageId: "message-1" }); });
    await clickConfirm(container);
    if (changed === "user") mocks.identity.userName = "users/other@example.com";
    else mocks.identity.workspaceName = "workspaces/other";
    await act(async () => finish());
    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    expect(mocks.markDraftCreated).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
  });

  test("does not redirect after the create page is unmounted while saving", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    let finish!: () => void;
    mocks.upsertPolicy.mockImplementationOnce(() => new Promise<void>(resolve => { finish = resolve; }));
    const { container, unmount } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context, { draftId: "exemption-1", messageId: "message-1" }); });
    await clickConfirm(container);
    unmount();
    await act(async () => finish());
    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    expect(mocks.markDraftCreated).toHaveBeenCalledExactlyOnceWith("chat", "message-1", "exemption-1");
    expect(mocks.routerPush).not.toHaveBeenCalled();
  });

  test("retains the unsaved form and allows retry after a failed save", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    mocks.upsertPolicy.mockRejectedValueOnce(new Error("Save failed"));
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context); });

    await clickConfirm(container);
    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    expect(isNavigationGuarded()).toBe(true);
    expect(await getMaskingExemptionEditorSnapshot(context)).toMatchObject({ members: draft.members, description: draft.description });

    await clickConfirm(container);
    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(2);
    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(mocks.routerPush).toHaveBeenCalledOnce();
    expect(isNavigationGuarded()).toBe(false);
  });

  test("keeps changes made during a pending save dirty after it succeeds", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "Review access", expirationTimestamp: null };
    let finish!: () => void;
    mocks.upsertPolicy.mockImplementationOnce(() => new Promise<void>(resolve => { finish = resolve; }));
    const { container } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context); });
    await clickConfirm(container);
    expect(isNavigationGuarded()).toBe(true);
    await click(container.querySelector<HTMLElement>('[data-testid="account-multi-select"]')!);
    await act(async () => finish());

    expect(mocks.upsertPolicy.mock.calls[0][0].policy.policy.value.exemptions[0].members).toEqual(draft.members);
    expect(await getMaskingExemptionEditorSnapshot(context)).toMatchObject({ members: [...draft.members, "users/alice"] });
    expect(isNavigationGuarded()).toBe(true);
    expect(mocks.routerBack).not.toHaveBeenCalled();
    expect(mocks.routerPush).not.toHaveBeenCalled();
    const confirm = Array.from(container.querySelectorAll<HTMLButtonElement>("button")).find(button => button.textContent === "common.confirm")!;
    expect(confirm.disabled).toBe(false);
  });

  test("does not overwrite dirty forms or a revision edited during async parsing", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: 'resource.column_name == "email"', description: "Draft", expirationTimestamp: null };
    const { container, unmount } = render();
    await click(container.querySelector<HTMLElement>('[data-testid="account-multi-select"]')!);
    await expect(openMaskingExemptionDraftInEditor(draft, context)).rejects.toThrow();
    const snapshot = await getMaskingExemptionEditorSnapshot(context);
    expect(snapshot).toBeTruthy();
    let finish!: (value: unknown) => void;
    mocks.parseDraftCondition.mockReturnValue(new Promise((resolve) => { finish = resolve; }));
    const pending = applyMaskingExemptionDraftInEditor(draft, snapshot!, context);
    await flush();
    await click(container.querySelector<HTMLElement>('[data-testid="account-multi-select"]')!);
    finish({ type: "ConditionGroup", args: ["draft"] });
    await expect(pending).rejects.toThrow();
    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    unmount();
  });
  test("uses ALL intentionally for an explicit project-wide true condition", async () => {
    const context = { workspaceName: "workspaces/default", userName: "users/member@example.com", chatId: "chat" };
    const draft = { project: "projects/proj1", scope: { kind: "project" as const }, members: ["user:member@example.com"], condition: "true", description: "All project columns", expirationTimestamp: null };
    const { container, unmount } = render();
    await act(async () => { await openMaskingExemptionDraftInEditor(draft, context); });
    expect(container.querySelectorAll('[role="radio"]')[0]).toHaveAttribute("aria-checked", "true");
    expect(await getMaskingExemptionEditorSnapshot(context)).toMatchObject({ members: draft.members, radioValue: "ALL", expirationTimestamp: null });
    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    unmount();
  });

  test("discovers special accounts in the workspace and project scopes", () => {
    const { unmount } = render();

    expect(mocks.accountSelectProps.value?.accountParents).toEqual([
      "workspaces/default",
      "projects/proj1",
    ]);

    unmount();
  });

  test("SELECT mode: wraps the OR-group so the expiration binds to every resource", async () => {
    const { container, unmount } = render();
    await flush();

    await selectResourceMode(container, "SELECT");
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-two-resources"]')!
    );
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expiration"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(1);
    // FIXED form: the OR-group is parenthesized and the time constraint leads.
    expect(submittedExpression()).toBe(`${TIME} && ((${c1}) || (${c2}))`);

    unmount();
  });

  test("SELECT mode without expiration: OR-group only, still parenthesized", async () => {
    const { container, unmount } = render();
    await flush();

    await selectResourceMode(container, "SELECT");
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-two-resources"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(submittedExpression()).toBe(`((${c1}) || (${c2}))`);

    unmount();
  });

  test("EXPRESSION mode: a top-level || in custom CEL is wrapped and bound by the expiration", async () => {
    mocks.buildCELExpr.mockResolvedValue({ parsed: true });
    mocks.batchConvertParsedExprToCELString.mockResolvedValue([
      'resource.database_name == "a" || resource.database_name == "b"',
    ]);

    const { container, unmount } = render();
    await flush();

    await selectResourceMode(container, "EXPRESSION");
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expr"]')!
    );
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expiration"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(submittedExpression()).toBe(
      `${TIME} && (resource.database_name == "a" || resource.database_name == "b")`
    );

    unmount();
  });

  test("ALL mode with expiration: time-only expression, no resource group", async () => {
    const { container, unmount } = render();
    await flush();

    // Default mode is ALL.
    await click(
      container.querySelector<HTMLElement>('[data-testid="set-expiration"]')!
    );
    await click(
      container.querySelector<HTMLElement>(
        '[data-testid="account-multi-select"]'
      )!
    );

    await clickConfirm(container);
    await flush();

    expect(submittedExpression()).toBe(TIME);

    unmount();
  });
});
