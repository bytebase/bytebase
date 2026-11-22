import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { Code, ConnectError } from "@connectrpc/connect";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { v4 as uuidv4 } from "uuid";
import type { AgentPageHelpRequest } from "@/modules/agent/logic/contextualHelp";

const {
  state,
  guard,
  fixtures,
  parsedExpr,
  parseCondition,
  validateCondition,
  parseDraftCondition,
  validateDraft,
  freshSemanticSetting,
  convertCondition,
  agentState,
  openAgentPrompt,
  openPageHelp,
  revokePageHelp,
} = vi.hoisted(() => ({
  fixtures: [
    {
      id: "rule-1",
      semanticType: "DEFAULT",
      condition: {
        title: "Test masking",
        expression: 'resource.environment_id in ["test"]',
        description: "Existing description",
        location: "",
      },
    },
    {
      id: "rule-2",
      semanticType: "PARTIAL",
      condition: {
        title: "Production masking",
        expression: 'resource.instance_id in ["production"]',
      },
    },
  ],
  parsedExpr: { type: "ConditionGroup", operator: "_&&_", args: [] },
  parseCondition: vi.fn<(...args: unknown[]) => Promise<unknown[]>>(),
  validateCondition: vi.fn(() => true),
  parseDraftCondition: vi.fn<(...args: unknown[]) => Promise<unknown>>(),
  validateDraft: vi.fn<(...args: unknown[]) => Promise<unknown>>(async (draft) => draft),
  freshSemanticSetting: vi.fn(async () => ({ value: { value: { case: "semanticType", value: { types: [] } } } })),
  convertCondition: vi.fn(async () => ['resource.environment_id in ["test"]']),
  agentState: {
    chats: [
      {
        id: "chat-1",
        status: "idle",
        workspaceName: "workspaces/default",
        archived: false,
      },
    ],
    currentChatId: "chat-1",
    getChat:
      vi.fn<
        (id: string) => {
          id: string;
          status: string;
          workspaceName: string;
          archived: boolean;
        } | null
      >(),
    pendingAskByChatId: {} as Record<string, unknown>,
    queuedUserMessagesByChatId: {} as Record<string, unknown>,
    queuedUserMessageInFlightByChatId: {} as Record<string, boolean>,
    abortControllersByChatId: {} as Record<string, unknown>,
    setCurrentChat: vi.fn(),
    queueUserMessage: vi.fn(),
    open: vi.fn(),
  },
  openAgentPrompt: vi.fn(
    (
      _prompt: string,
      _resources?: unknown,
      onCreated?: (id: string) => void
    ) => {
      onCreated?.("manual-chat");
    }
  ),
  openPageHelp: vi.fn<(options: Omit<AgentPageHelpRequest, "workspaceName" | "userName">) => () => void>(),
  revokePageHelp: vi.fn(),
  guard: vi.fn(),
  state: {
    environmentList: [],
    getOrFetchPolicyByParentAndType: vi.fn(),
    fetchPolicyByParentAndType: vi.fn(),
    getOrFetchSettingByName: vi.fn(async () => undefined),
    setSettingByName: vi.fn(),
    hasInstanceFeature: vi.fn(() => true),
    hasPermission: true,
    availableSemanticTypeIds: ["DEFAULT", "PARTIAL", "CUSTOM"],
    settingsByName: {},
    currentUser: { name: "users/member@example.com" },
    upsertPolicy: vi.fn(async (_request: unknown) => undefined),
    workspaceResourceName: () => "workspaces/default",
    refreshSubscription: vi.fn(async () => ({})),
  },
}));

vi.mock("@/modules/cel", () => ({
  ExprType: { RawString: "RawString", ConditionGroup: "ConditionGroup" },
  buildCELExpr: vi.fn(async () => ({})),
  emptySimpleExpr: () => parsedExpr,
  resolveCELExpr: () => parsedExpr,
  validateSimpleExpr: validateCondition,
  wrapAsGroup: (expr: unknown) => expr,
}));
vi.mock("@/modules/agent/logic/maskingRuleDraft", () => ({
  parseMaskingRuleDraftCondition: parseDraftCondition,
  validateMaskingRuleDraft: validateDraft,
}));
vi.mock("@/api", () => ({ settingServiceClientConnect: { getSetting: freshSemanticSetting } }));
vi.mock("@/app/router", () => ({
  router: {
    currentRoute: {
      value: { fullPath: "/settings/sensitive-data/global-masking" },
    },
  },
}));
vi.mock("@/modules/agent/store/agent", () => ({
  useAgentStore: Object.assign(
    (selector: (store: typeof agentState) => unknown) => selector(agentState),
    { getState: () => agentState, subscribe: vi.fn(() => () => {}) }
  ),
}));
vi.mock("@/modules/agent/logic/instanceConnectionHelp", () => ({
  openAgentWithPrompt: openAgentPrompt,
}));
vi.mock("@/modules/agent/logic/contextualHelp", () => ({
  openAgentPageHelp: openPageHelp,
}));
vi.mock("@/components/ExprEditor", () => ({
  ExprEditor: ({
    expr,
    readonly,
    onUpdate,
    bordered = true,
  }: {
    expr: unknown;
    readonly?: boolean;
    onUpdate: (expr: unknown) => void;
    bordered?: boolean;
  }) => (
    <>
      <output
        data-testid="condition-editor"
        data-readonly={readonly}
        data-bordered={bordered}
        data-agent-help={!readonly}
      >
        {JSON.stringify(expr)}
      </output>
      {!readonly && (
        <>
          <button
            type="button"
            onClick={() =>
              onUpdate({
                ...parsedExpr,
                args: [
                  {
                    type: "Condition",
                    operator: "_==_",
                    args: ["resource.column_name", "email"],
                  },
                ],
              })
            }
          >
            Change conditions
          </button>
          <button
            type="button"
            onClick={() =>
              onUpdate({
                ...parsedExpr,
                args: [
                  { type: "RawString", content: "resource.column_name == " },
                ],
              })
            }
          >
            Incomplete raw condition
          </button>
        </>
      )}
    </>
  ),
}));
vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));
vi.mock("@/components/SemanticTypeSelect", () => ({
  SemanticTypeOption: ({
    semanticType,
    descriptionAsTooltip,
  }: {
    semanticType: { title: string; description: string };
    descriptionAsTooltip?: boolean;
  }) => (
    <span>
      {semanticType.title}
      {!descriptionAsTooltip && <span>{semanticType.description}</span>}
    </span>
  ),
  SemanticTypeSelect: ({
    value,
    onValueChange,
    disabled,
    className,
    id,
    "aria-label": label,
    "aria-describedby": describedBy,
    "aria-invalid": invalid,
  }: {
    value: string;
    onValueChange: (value: string) => void;
    disabled?: boolean;
    className?: string;
    id?: string;
    "aria-label"?: string;
    "aria-describedby"?: string;
    "aria-invalid"?: boolean;
  }) => (
    <select
      id={id}
      className={className}
      aria-label={label ?? "Semantic type"}
      aria-describedby={describedBy}
      aria-invalid={invalid}
      value={value}
      disabled={disabled}
      onChange={(event) => onValueChange(event.target.value)}
    >
      <option value="">Select semantic type</option>
      <option value="DEFAULT">Default</option>
      <option value="PARTIAL">Default Partial</option>
      <option value="CUSTOM">Custom</option>
    </select>
  ),
}));
vi.mock("@/hooks/useSemanticTypes", () => ({
  useSemanticTypes: () => ({
    semanticTypes: [
      { id: "DEFAULT", title: "Default", description: "Mask all values" },
      {
        id: "PARTIAL",
        title: "Default Partial",
        description: "Mask part of each value",
      },
      { id: "CUSTOM", title: "Custom", description: "Custom masking" },
    ].filter((type) => state.availableSemanticTypeIds.includes(type.id)),
  }),
}));
vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: guard,
}));
vi.mock("@/lib/sensitive-data/components-utils", () => ({
  getClassificationLevelOptions: () => [],
}));
vi.mock("@/stores/app", () => ({
  useAppStore: Object.assign(
    (selector: (store: typeof state) => unknown) => selector(state),
    { getState: () => state }
  ),
}));
vi.mock("@/stores", () => ({ pushNotification: vi.fn() }));
vi.mock("@/types/proto-es/google/type/expr_pb", () => ({ ExprSchema: {} }));
vi.mock("@/types/proto-es/v1/org_policy_service_pb", () => ({
  MaskingRulePolicy_MaskingRuleSchema: {},
  MaskingRulePolicySchema: {},
  PolicyResourceType: { WORKSPACE: 1 },
  PolicyType: { MASKING_RULE: 1 },
}));
vi.mock("@/types/proto-es/v1/setting_service_pb", () => ({
  GetSettingRequestSchema: {},
  SettingSchema: {},
  Setting_SettingName: { DATA_CLASSIFICATION: 2, SEMANTIC_TYPES: 1 },
}));
vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_DATA_MASKING: 1 },
}));
vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));
vi.mock("uuid", () => ({ v4: vi.fn(() => "new-rule") }));
vi.mock("@/utils", () => ({
  arraySwap: (items: unknown[], a: number, b: number) => {
    [items[a], items[b]] = [items[b], items[a]];
  },
  batchConvertCELStringToParsedExpr: parseCondition,
  batchConvertParsedExprToCELString: convertCondition,
  getDatabaseIdOptionConfig: () => ({ options: [] }),
  getInstanceIdOptionConfig: () => ({ options: [] }),
  getProjectIdOptionConfig: () => ({ options: [] }),
  hasWorkspacePermissionV2: () => state.hasPermission,
}));
vi.mock("react-i18next", () => ({
  initReactI18next: { type: "3rdParty", init: () => {} },
  useTranslation: () => ({ t: (key: string) => key }),
}));

import { GlobalMaskingPage } from "./GlobalMaskingPage";
import {
  applyMaskingRuleDraftInEditor,
  getMaskingRuleEditorSnapshot,
  isMaskingRuleDraftTargetReady,
  openMaskingRuleDraftInEditor,
  openMaskingRuleDraftsInEditor,
} from "@/modules/agent/logic/maskingRuleDraftTarget";
const prefix = "settings.sensitive-data.global-rules.";
const agentContext = {
  workspaceName: "workspaces/default",
  userName: "users/member@example.com",
  chatId: "chat-1",
};
const agentDraft = {
  title: "Agent email masking",
  semanticType: "DEFAULT",
  condition: 'resource.environment_id in ["test"]',
  scope: { kind: "workspace" as const },
};

beforeEach(() => {
  vi.clearAllMocks();
  openPageHelp.mockReturnValue(revokePageHelp);
  parseDraftCondition.mockReset().mockResolvedValue({
    ...parsedExpr,
    args: [
      {
        type: "Condition",
        operator: "_==_",
        args: ["resource.column_name", "email"],
      },
    ],
  });
  convertCondition.mockReset().mockResolvedValue([agentDraft.condition]);
  agentState.chats = [
    {
      id: "chat-1",
      status: "idle",
      workspaceName: "workspaces/default",
      archived: false,
    },
  ];
  agentState.currentChatId = "chat-1";
  state.currentUser = { name: "users/member@example.com" };
  freshSemanticSetting.mockReset().mockResolvedValue({ value: { value: { case: "semanticType", value: { types: [] } } } });
  validateDraft.mockReset().mockImplementation(async (draft) => draft);
  agentState.getChat.mockImplementation(
    (id) => agentState.chats.find((chat) => chat.id === id) ?? null
  );
  agentState.setCurrentChat.mockImplementation((id: string) => {
    agentState.currentChatId = id;
  });
  openAgentPrompt.mockImplementation((_prompt, _resources, onCreated) => {
    agentState.chats.push({
      id: "manual-chat",
      status: "idle",
      workspaceName: "workspaces/default",
      archived: false,
    });
    agentState.currentChatId = "manual-chat";
    onCreated?.("manual-chat");
  });
  agentState.pendingAskByChatId = {};
  agentState.queuedUserMessagesByChatId = {};
  agentState.queuedUserMessageInFlightByChatId = {};
  agentState.abortControllersByChatId = {};
  parseCondition
    .mockReset()
    .mockResolvedValue([{ exprKind: { case: "callExpr" } }]);
  state.hasPermission = true;
  state.availableSemanticTypeIds = ["DEFAULT", "PARTIAL", "CUSTOM"];
  validateCondition.mockReturnValue(true);
  state.hasInstanceFeature.mockReturnValue(true);
  state.upsertPolicy.mockReset().mockResolvedValue(undefined);
  state.getOrFetchPolicyByParentAndType.mockResolvedValue({
    policy: {
      case: "maskingRulePolicy",
      value: { rules: structuredClone(fixtures) },
    },
  });
  state.fetchPolicyByParentAndType
    .mockReset()
    .mockImplementation((request) =>
      state.getOrFetchPolicyByParentAndType(request)
    );
});
afterEach(cleanup);
async function renderPage() {
  render(<GlobalMaskingPage />);
  await screen.findByText("Test masking");
  await waitFor(() =>
    expect(isMaskingRuleDraftTargetReady(agentContext)).toBe(
      state.hasPermission && state.hasInstanceFeature()
    )
  );
}
async function editFirstRule() {
  await renderPage();
  fireEvent.click(
    screen.getAllByRole("button", { name: prefix + "edit-rule" })[0]
  );
  return await screen.findByRole("textbox", { name: prefix + "name" });
}
async function createDraftForm() {
  fireEvent.click(screen.getByRole("button", { name: prefix + "create-rule" }));
  const name = await screen.findByRole("textbox", { name: prefix + "name" });
  await waitFor(() => expect(name).toBeEnabled());
  fireEvent.change(name, { target: { value: "Manual draft" } });
  fireEvent.change(screen.getByRole("combobox"), {
    target: { value: "DEFAULT" },
  });
  fireEvent.click(screen.getByRole("button", { name: "Change conditions" }));
  return name;
}
async function createManualDraft() {
  await renderPage();
  return createDraftForm();
}
function savedRules() {
  const request = state.upsertPolicy.mock.calls.at(-1)?.[0] as {
    policy: { policy: { value: { rules: typeof fixtures } } };
  };
  return request.policy.policy.value.rules;
}
function openRuleHelp() {
  fireEvent.click(screen.getByRole("button", { name: "agent.masking.ask-agent-short" }));
  return openPageHelp.mock.calls.at(-1)![0];
}

describe("GlobalMaskingPage", () => {
  it("opens and saves both queued rules when no semantic type setting exists", async () => {
    freshSemanticSetting.mockRejectedValue(
      new ConnectError("Setting not found", Code.NotFound)
    );
    await renderPage();
    state.upsertPolicy.mockImplementation(async (request) => {
      const { policy } = request as { policy: { policy: unknown } };
      state.getOrFetchPolicyByParentAndType.mockResolvedValue(policy);
    });

    await act(async () =>
      openMaskingRuleDraftsInEditor(
        [agentDraft, { ...agentDraft, title: "Name masking", semanticType: "PARTIAL" }],
        agentContext
      )
    );
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue(agentDraft.title);
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(1));
    fireEvent.click(await screen.findByRole("button", { name: "agent.masking-rule-draft.queue.review" }));
    await screen.findByDisplayValue("Name masking");
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(2));

    expect(savedRules().slice(0, 2)).toEqual(fixtures);
    expect(savedRules().slice(2).map((rule) => rule.semanticType)).toEqual(["DEFAULT", "PARTIAL"]);
  });

  it.each([Code.PermissionDenied, Code.Unavailable])(
    "preserves queued drafts and existing rules after semantic setting read failure %s",
    async (code) => {
      await renderPage();
      await act(async () =>
        openMaskingRuleDraftsInEditor(
          [agentDraft, { ...agentDraft, title: "Second draft" }],
          agentContext
        )
      );
      freshSemanticSetting.mockRejectedValue(new ConnectError("Read failed", code));
      fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);

      await screen.findByText(prefix + "error.save-failed");
      expect(state.upsertPolicy).not.toHaveBeenCalled();
      expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue(agentDraft.title);
      expect(screen.getByText("Test masking")).toBeInTheDocument();
      expect(screen.getByText("Production masking")).toBeInTheDocument();
    }
  );

  it.each(["undefined", "rejected", "incompatible"])(
    "preserves existing rules when a %s policy read cannot verify a recheck",
    async (failure) => {
      await renderPage();
      await act(async () =>
        openMaskingRuleDraftsInEditor(
          [agentDraft, { ...agentDraft, title: "Second draft" }],
          agentContext
        )
      );
      freshSemanticSetting.mockResolvedValue({
        value: {
          value: {
            case: "semanticType",
            value: { types: [{ id: "changed" }] },
          },
        },
      } as never);
      fireEvent.click(
        screen
          .getAllByRole("button", { name: prefix + "create-rule" })
          .at(-1)!
      );
      const recheck = await screen.findByRole("button", {
        name: "agent.masking-rule-draft.queue.recheck",
      });
      if (failure === "rejected") {
        state.getOrFetchPolicyByParentAndType.mockRejectedValue(
          new Error("Policy read failed")
        );
      } else {
        state.getOrFetchPolicyByParentAndType.mockResolvedValue(
          failure === "undefined"
            ? undefined
            : { policy: { case: undefined } }
        );
      }
      await act(async () => fireEvent.click(recheck));

      expect(screen.getByText("Test masking")).toBeInTheDocument();
      expect(screen.getByText("Production masking")).toBeInTheDocument();
      expect(recheck).toBeInTheDocument();
      expect(screen.getByText(prefix + "error.save-failed")).toBeInTheDocument();
      expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue(
        agentDraft.title
      );
      await act(async () =>
        fireEvent.click(
          screen
            .getAllByRole("button", { name: prefix + "create-rule" })
            .at(-1)!
        )
      );
      expect(state.upsertPolicy).not.toHaveBeenCalled();

      state.getOrFetchPolicyByParentAndType.mockResolvedValue({
        policy: {
          case: "maskingRulePolicy",
          value: { rules: structuredClone(fixtures) },
        },
      });
      await act(async () => fireEvent.click(recheck));
      await act(async () =>
        fireEvent.click(
          screen
            .getAllByRole("button", { name: prefix + "create-rule" })
            .at(-1)!
        )
      );
      expect(savedRules().slice(0, 2)).toEqual(fixtures);
      expect(savedRules()).toHaveLength(3);
    }
  );

  it("creates a queued rule when the authoritative read confirms no policy exists", async () => {
    state.getOrFetchPolicyByParentAndType.mockResolvedValue(undefined);
    state.fetchPolicyByParentAndType.mockResolvedValue(null);
    render(<GlobalMaskingPage />);
    await waitFor(() =>
      expect(isMaskingRuleDraftTargetReady(agentContext)).toBe(true)
    );
    await act(async () =>
      openMaskingRuleDraftsInEditor(
        [agentDraft, { ...agentDraft, title: "Second draft" }],
        agentContext
      )
    );
    await act(async () =>
      fireEvent.click(
        screen
          .getAllByRole("button", { name: prefix + "create-rule" })
          .at(-1)!
      )
    );

    expect(savedRules()).toHaveLength(1);
    expect(savedRules()[0]).toMatchObject({
      semanticType: "DEFAULT",
      condition: { title: agentDraft.title },
    });
  });

  it.each(["save", "cancel"])(
    "keeps completed drafts out of native %s handling after the Agent chat changes",
    async (action) => {
      await renderPage();
      state.upsertPolicy.mockImplementation(async (request) => {
        const { policy } = request as { policy: { policy: unknown } };
        state.getOrFetchPolicyByParentAndType.mockResolvedValue(policy);
      });
      await act(async () =>
        openMaskingRuleDraftsInEditor(
          [agentDraft, { ...agentDraft, title: "Second draft" }],
          agentContext
        )
      );
      await act(async () =>
        fireEvent.click(
          screen
            .getAllByRole("button", { name: prefix + "create-rule" })
            .at(-1)!
        )
      );
      await act(async () =>
        fireEvent.click(
          screen.getByRole("button", {
            name: "agent.masking-rule-draft.queue.review",
          })
        )
      );
      await screen.findByDisplayValue("Second draft");
      await act(async () =>
        fireEvent.click(
          screen
            .getAllByRole("button", { name: prefix + "create-rule" })
            .at(-1)!
        )
      );
      expect(savedRules()).toHaveLength(4);
      agentState.currentChatId = "another-chat";
      fireEvent.click(
        screen.getAllByRole("button", { name: prefix + "edit-rule" })[2]
      );
      const name = await screen.findByRole("textbox", { name: prefix + "name" });
      await waitFor(() => expect(name).toBeEnabled());
      fireEvent.change(name, { target: { value: "Native title edit" } });
      await act(async () =>
        fireEvent.click(
          screen.getByRole("button", {
            name: action === "save" ? prefix + "save-rule" : "common.cancel",
          })
        )
      );

      expect(
        screen.queryByRole("textbox", { name: prefix + "name" })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", {
          name: "agent.masking-rule-draft.queue.review",
        })
      ).not.toBeInTheDocument();
      expect(
        screen.getAllByRole("button", { name: prefix + "edit-rule" })[2]
      ).toBeEnabled();
      expect(guard).toHaveBeenLastCalledWith(false);
      expect(savedRules().slice(0, 2)).toEqual(fixtures);
      expect(savedRules()[2].condition.title).toBe(
        action === "save" ? "Native title edit" : agentDraft.title
      );
      expect(savedRules()).toHaveLength(4);
    }
  );

  it("stages a batch atomically and retains incomplete edits when switching", async () => {
    await renderPage();
    await act(async () => openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Second draft" }], agentContext));
    const name = screen.getByRole("textbox", { name: prefix + "name" });
    fireEvent.change(name, { target: { value: "Edited first draft" } });
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "PARTIAL" } });
    fireEvent.click(screen.getByRole("button", { name: "Incomplete raw condition" }));
    fireEvent.click(screen.getAllByRole("button", { name: "agent.masking-rule-draft.queue.review" })[0]);
    await waitFor(() => expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue("Second draft"));
    fireEvent.click(screen.getAllByRole("button", { name: "agent.masking-rule-draft.queue.review" })[0]);
    await waitFor(() => expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue("Edited first draft"));
    expect(screen.getByRole("combobox")).toHaveValue("PARTIAL");
    expect(screen.getAllByTestId("condition-editor").find((item) => item.dataset.readonly !== "true")).toHaveTextContent("resource.column_name == ");
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("retains an applied draft's replacement scope across queue switches without changing other drafts", async () => {
    await renderPage();
    const scopeA = { kind: "project" as const, project: "projects/scope-a" };
    const scopeB = { kind: "project" as const, project: "projects/scope-b" };
    const conditionA = 'resource.project_id == "scope-a" && resource.column_name == "email"';
    const conditionB = 'resource.project_id == "scope-b" && resource.column_name == "email"';
    validateDraft.mockImplementation(async (draft) => {
      const proposal = draft as { condition: string; scope: { project: string } };
      const required = `resource.project_id == "${proposal.scope.project.slice("projects/".length)}"`;
      return { ...proposal, condition: proposal.condition.includes(required) ? proposal.condition : `${required} && (${proposal.condition})` };
    });
    await act(async () => openMaskingRuleDraftsInEditor([
      { ...agentDraft, condition: conditionA, scope: scopeA },
      { ...agentDraft, title: "Unchanged second draft", condition: conditionA, scope: scopeA },
    ], agentContext));
    const queue = screen.getByRole("region", { name: "agent.masking-rule-draft.queue.title" });
    expect(within(queue).getAllByText(scopeA.project)).toHaveLength(2);
    const snapshot = await getMaskingRuleEditorSnapshot(agentContext);
    convertCondition.mockResolvedValue([conditionB]);
    await act(async () => applyMaskingRuleDraftInEditor({ ...agentDraft, title: "Replacement draft", condition: conditionB, scope: scopeB }, snapshot!, agentContext));
    expect(within(queue).getAllByText(scopeA.project)).toHaveLength(1);
    expect(within(queue).getByText(scopeB.project)).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "agent.masking-rule-draft.queue.review" }));
    await screen.findByDisplayValue("Unchanged second draft");
    fireEvent.click(screen.getByRole("button", { name: "agent.masking-rule-draft.queue.review" }));
    await screen.findByDisplayValue("Replacement draft");
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(1));
    expect(savedRules().at(-1)?.condition.expression).toBe(conditionB);
    expect(savedRules().at(-1)?.condition.expression).not.toContain("scope-a");
    const secondDraftChecks = validateDraft.mock.calls.map(([draft]) => draft as { title: string; scope: unknown }).filter((draft) => draft.title === "Unchanged second draft");
    expect(secondDraftChecks.length).toBeGreaterThan(0);
    expect(secondDraftChecks.every((draft) => JSON.stringify(draft.scope) === JSON.stringify(scopeA))).toBe(true);
    expect(screen.getByText("Unchanged second draft")).toBeInTheDocument();
  });

  it("saves one queued draft per click, retains failed saves, and does not select the next", async () => {
    await renderPage();
    await act(async () => openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Second draft" }], agentContext));
    state.upsertPolicy.mockRejectedValueOnce(new Error("write failed"));
    const create = () => screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!;
    fireEvent.click(create());
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(create()).toBeEnabled());
    fireEvent.click(create());
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(2));
    await waitFor(() => expect(screen.queryByRole("textbox", { name: prefix + "name" })).not.toBeInTheDocument());
    expect(savedRules()).toHaveLength(fixtures.length + 1);
    expect(screen.getByText("agent.masking-rule-draft.queue.created")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "agent.masking-rule-draft.queue.review" })).toBeInTheDocument();
  });

  it.each(["policy", "semantic"])("requires explicit recheck after external %s drift and preserves edits", async (kind) => {
    await renderPage();
    await act(async () => openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Second draft" }], agentContext));
    fireEvent.change(screen.getByRole("textbox", { name: prefix + "name" }), { target: { value: "Preserved edits" } });
    if (kind === "policy") state.getOrFetchPolicyByParentAndType.mockResolvedValue({ policy: { case: "maskingRulePolicy", value: { rules: [...fixtures, { ...fixtures[0], id: "external", condition: { ...fixtures[0].condition, title: "External rule" } }] } } });
    else freshSemanticSetting.mockResolvedValue({ value: { value: { case: "semanticType", value: { types: [{ id: "changed" }] } } } } as never);
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    const recheck = await screen.findByRole("button", { name: "agent.masking-rule-draft.queue.recheck" });
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue("Preserved edits");
    fireEvent.click(recheck);
    await waitFor(() => expect(screen.queryByRole("button", { name: "agent.masking-rule-draft.queue.recheck" })).not.toBeInTheDocument());
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue("Preserved edits");
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(1));
    expect(savedRules()).toHaveLength(fixtures.length + (kind === "policy" ? 2 : 1));
  });

  it("rejects the entire batch when a later expression cannot load", async () => {
    await renderPage();
    parseDraftCondition.mockResolvedValueOnce(parsedExpr).mockRejectedValueOnce(new Error("invalid condition"));
    await act(async () => {
      await expect(openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Bad draft" }], agentContext)).rejects.toThrow("invalid condition");
    });
    expect(screen.queryByRole("textbox", { name: prefix + "name" })).not.toBeInTheDocument();
    expect(screen.queryByText("agent.masking-rule-draft.queue.title")).not.toBeInTheDocument();
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("does not save a queued draft after the page unmounts during validation", async () => {
    await renderPage();
    await act(async () => openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Second draft" }], agentContext));
    let resolve!: (draft: unknown) => void;
    validateDraft.mockImplementationOnce(() => new Promise((done) => { resolve = done; }));
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(resolve).toBeTypeOf("function"));
    cleanup();
    await act(async () => { resolve(agentDraft); });
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("keeps the expected baseline current after its own save", async () => {
    await renderPage();
    state.upsertPolicy.mockImplementation(async (request) => {
      const { policy } = request as { policy: { policy: unknown } };
      state.getOrFetchPolicyByParentAndType.mockResolvedValue(policy);
    });
    await act(async () => openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Second draft" }], agentContext));
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(screen.queryByRole("textbox", { name: prefix + "name" })).not.toBeInTheDocument());
    fireEvent.click(screen.getByRole("button", { name: "agent.masking-rule-draft.queue.review" }));
    await waitFor(() => expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue("Second draft"));
    expect(screen.queryByRole("button", { name: "agent.masking-rule-draft.queue.recheck" })).not.toBeInTheDocument();
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(2));
    expect(savedRules()).toHaveLength(fixtures.length + 2);
  });

  it("continues valid drafts after recheck and permits repairing a removed pending semantic type", async () => {
    await renderPage();
    state.upsertPolicy.mockImplementation(async (request) => {
      const { policy } = request as { policy: { policy: unknown } };
      state.getOrFetchPolicyByParentAndType.mockResolvedValue(policy);
    });
    await act(async () => openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Custom draft", semanticType: "CUSTOM" }], agentContext));
    state.availableSemanticTypeIds = ["DEFAULT", "PARTIAL"];
    freshSemanticSetting.mockResolvedValue({ value: { value: { case: "semanticType", value: { types: [{ id: "remaining" }] } } } } as never);
    validateDraft.mockImplementation(async (draft, _context, options) => {
      const proposal = draft as typeof agentDraft;
      if (proposal.semanticType === "CUSTOM" && !(options as { purpose?: string } | undefined)?.purpose) throw new Error("Semantic type removed");
      return draft;
    });
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    fireEvent.click(await screen.findByRole("button", { name: "agent.masking-rule-draft.queue.recheck" }));
    await waitFor(() => expect(screen.queryByRole("button", { name: "agent.masking-rule-draft.queue.recheck" })).not.toBeInTheDocument());
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(screen.queryByRole("textbox", { name: prefix + "name" })).not.toBeInTheDocument());
    expect(state.upsertPolicy).toHaveBeenCalledTimes(1);
    fireEvent.click(screen.getByRole("button", { name: "agent.masking-rule-draft.queue.review" }));
    await screen.findByDisplayValue("Custom draft");
    expect(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!).toBeDisabled();
    expect(screen.getByText(prefix + "error.missing-semantic-type")).toBeInTheDocument();
    fireEvent.change(screen.getByRole("combobox"), { target: { value: "PARTIAL" } });
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(2));
    expect(savedRules().at(-1)?.semanticType).toBe("PARTIAL");
  });

  it.each(["user", "chat", "permission", "feature"])("blocks queued saves after %s changes", async (changed) => {
    await renderPage();
    await act(async () => openMaskingRuleDraftsInEditor([agentDraft, { ...agentDraft, title: "Second draft" }], agentContext));
    if (changed === "user") state.currentUser = { name: "users/other@example.com" };
    else if (changed === "chat") agentState.currentChatId = "other-chat";
    else if (changed === "permission") state.hasPermission = false;
    else state.hasInstanceFeature.mockReturnValue(false);
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "create-rule" }).at(-1)!);
    await screen.findByText(prefix + "error.save-failed");
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue(agentDraft.title);
  });
  it("opens page help without creating or submitting a chat", async () => {
    await renderPage();
    expect(openRuleHelp()).toEqual({ page: { kind: "global-masking" } });
    expect(openAgentPrompt).not.toHaveBeenCalled();
    expect(agentState.queueUserMessage).not.toHaveBeenCalled();
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });
  it("keeps the toolbar action and expression Agent help across create and edit states", async () => {
    await renderPage();
    const help = screen.getByRole("button", { name: "agent.masking.ask-agent-short" });
    expect(help).toHaveClass("h-9", "text-sm", "text-accent");
    expect(help).not.toHaveClass("h-7", "border");
    expect(screen.getAllByRole("button", { name: /agent.*(?:ask-agent|review)/i })).toHaveLength(1);
    fireEvent.click(screen.getByRole("button", { name: prefix + "create-rule" }));
    const name = await screen.findByRole("textbox", { name: prefix + "name" });
    await waitFor(() => expect(name).toBeEnabled());
    expect(screen.getByRole("button", { name: "agent.masking.ask-agent-short" })).toBe(help);
    expect(screen.getAllByTestId("condition-editor").find((editor) => editor.dataset.readonly !== "true")).toHaveAttribute("data-agent-help", "true");
    expect(screen.queryByRole("button", { name: "agent.masking-rule-draft.review" })).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "edit-rule" })[0]);
    await waitFor(() => expect(screen.getByRole("textbox", { name: prefix + "name" })).toBeEnabled());
    expect(screen.getByRole("button", { name: "agent.masking.ask-agent-short" })).toBe(help);
    expect(screen.getAllByTestId("condition-editor").find((editor) => editor.dataset.readonly !== "true")).toHaveAttribute("data-agent-help", "true");
    expect(screen.queryByRole("button", { name: "agent.masking-rule-draft.review" })).not.toBeInTheDocument();
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });
  it("opens an Agent draft locally and persists only after Create rule", async () => {
    await renderPage();
    await act(async () => {
      await openMaskingRuleDraftInEditor(agentDraft, agentContext);
    });
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue(
      agentDraft.title
    );
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveFocus();
    expect(screen.getByRole("combobox")).toHaveValue("DEFAULT");
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    expect(guard).toHaveBeenLastCalledWith(true);
    const editor = screen.getAllByRole("listitem").at(-1)!;
    const create = within(editor).getByRole("button", {
      name: prefix + "create-rule",
    });
    expect(create).toBeEnabled();
    fireEvent.click(create);
    await screen.findByText(agentDraft.title);
    expect(savedRules().slice(0, 2)).toEqual(fixtures);
    expect(savedRules()[2]).toMatchObject({
      id: "new-rule",
      semanticType: "DEFAULT",
      condition: { title: agentDraft.title },
    });
    expect(await getMaskingRuleEditorSnapshot(agentContext)).toBeUndefined();
  });

  it.each(["create", "edit", "reorder"])(
    "never replaces an existing %s workflow",
    async (mode) => {
      await renderPage();
      fireEvent.click(
        mode === "edit"
          ? screen.getAllByRole("button", { name: prefix + "edit-rule" })[0]
          : screen.getByRole("button", {
              name: prefix + (mode === "create" ? "create-rule" : "re-order"),
            })
      );
      await expect(
        openMaskingRuleDraftInEditor(agentDraft, agentContext)
      ).rejects.toThrow();
      expect(parseDraftCondition).not.toHaveBeenCalled();
      expect(state.upsertPolicy).not.toHaveBeenCalled();
      if (mode === "edit") {
        expect(
          screen.getByRole("textbox", { name: prefix + "name" })
        ).toHaveValue(fixtures[0].condition.title);
      }
    }
  );

  it("applies only to the unchanged new draft and preserves existing rule order", async () => {
    const name = await createManualDraft();
    const original = await getMaskingRuleEditorSnapshot(agentContext);
    expect(original).toMatchObject({
      title: "Manual draft",
      condition: fixtures[0].condition.expression,
    });
    fireEvent.change(name, { target: { value: "Manual update" } });
    await act(async () => {
      await expect(
        applyMaskingRuleDraftInEditor(agentDraft, original!, agentContext)
      ).rejects.toThrow();
    });
    expect(name).toHaveValue("Manual update");
    const current = await getMaskingRuleEditorSnapshot(agentContext);
    await act(async () => {
      await applyMaskingRuleDraftInEditor(agentDraft, current!, agentContext);
    });
    expect(name).toHaveValue(agentDraft.title);
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    const applied = await getMaskingRuleEditorSnapshot(agentContext);
    expect(applied?.chatId).toBe(agentContext.chatId);
    await expect(
      applyMaskingRuleDraftInEditor(agentDraft, applied!, {
        ...agentContext,
        chatId: "chat-2",
      })
    ).rejects.toThrow();
    fireEvent.click(
      within(screen.getAllByRole("listitem").at(-1)!).getByRole("button", {
        name: prefix + "create-rule",
      })
    );
    await screen.findByText(agentDraft.title);
    expect(savedRules().map((rule) => rule.id)).toEqual([
      ...fixtures.map((rule) => rule.id),
      "new-rule",
    ]);
  });

  it("rejects invalid incoming selectors and keeps the editor unchanged", async () => {
    const name = await createManualDraft();
    const original = await getMaskingRuleEditorSnapshot(agentContext);
    parseDraftCondition.mockRejectedValueOnce(
      new Error("Unsupported selector")
    );
    await act(async () => {
      await expect(
        applyMaskingRuleDraftInEditor(agentDraft, original!, agentContext)
      ).rejects.toThrow("Unsupported selector");
    });
    expect(name).toHaveValue("Manual draft");
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("registers only when allowed and unregisters on leaving the page", async () => {
    state.hasPermission = false;
    const view = render(<GlobalMaskingPage />);
    await screen.findByText("Test masking");
    expect(isMaskingRuleDraftTargetReady(agentContext)).toBe(false);
    state.hasPermission = true;
    view.rerender(<GlobalMaskingPage />);
    await waitFor(() =>
      expect(isMaskingRuleDraftTargetReady(agentContext)).toBe(true)
    );
    view.unmount();
    expect(isMaskingRuleDraftTargetReady(agentContext)).toBe(false);
  });

  it("locks pending draft parsing and rejects a second open without adding two rules", async () => {
    let complete!: (expr: typeof parsedExpr) => void;
    parseDraftCondition.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          complete = resolve;
        })
    );
    await renderPage();
    let pending!: Promise<void>;
    await act(async () => {
      pending = openMaskingRuleDraftInEditor(agentDraft, agentContext);
    });
    expect(
      screen.getByRole("button", { name: prefix + "create-rule" })
    ).toBeDisabled();
    await expect(
      openMaskingRuleDraftInEditor(agentDraft, agentContext)
    ).rejects.toThrow();
    await act(async () => {
      complete(parsedExpr);
      await pending;
    });
    expect(screen.getAllByRole("listitem")).toHaveLength(3);
    expect(screen.getAllByRole("textbox")).toHaveLength(1);
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("discards pending handoff after the page unmounts", async () => {
    let complete!: (expr: typeof parsedExpr) => void;
    parseDraftCondition.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          complete = resolve;
        })
    );
    const view = render(<GlobalMaskingPage />);
    await screen.findByText("Test masking");
    await waitFor(() =>
      expect(isMaskingRuleDraftTargetReady(agentContext)).toBe(true)
    );
    const pending = openMaskingRuleDraftInEditor(agentDraft, agentContext);
    view.unmount();
    complete(parsedExpr);
    await expect(pending).rejects.toThrow();
    expect(isMaskingRuleDraftTargetReady(agentContext)).toBe(false);
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it.each([
    ["open", "chat"],
    ["apply", "chat"],
    ["open", "actor"],
    ["apply", "actor"],
    ["open", "workspace"],
    ["apply", "workspace"],
  ])(
    "rejects pending %s when the %s changes during selector parsing",
    async (operation, change) => {
      if (operation === "apply") await createManualDraft();
      else await renderPage();
      const snapshot =
        operation === "apply"
          ? await getMaskingRuleEditorSnapshot(agentContext)
          : undefined;
      let complete!: (expr: typeof parsedExpr) => void;
      parseDraftCondition.mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            complete = resolve;
          })
      );
      let pending!: Promise<void>;
      await act(async () => {
        pending =
          operation === "apply"
            ? applyMaskingRuleDraftInEditor(agentDraft, snapshot!, agentContext)
            : openMaskingRuleDraftInEditor(agentDraft, agentContext);
      });
      if (change === "chat") agentState.currentChatId = "chat-2";
      else if (change === "actor")
        state.currentUser = { name: "users/other@example.com" };
      else agentState.chats[0].workspaceName = "workspaces/other";
      await act(async () => {
        complete(parsedExpr);
        await expect(pending).rejects.toThrow();
      });
      expect(state.upsertPolicy).not.toHaveBeenCalled();
      if (operation === "apply")
        expect(
          screen.getByRole("textbox", { name: prefix + "name" })
        ).toHaveValue("Manual draft");
      else expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    }
  );

  it("rejects an apply after the original editor is canceled and reopened", async () => {
    vi.mocked(uuidv4 as () => string)
      .mockReturnValueOnce("old-rule")
      .mockReturnValueOnce("old-editor");
    await createManualDraft();
    const snapshot = await getMaskingRuleEditorSnapshot(agentContext);
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    vi.mocked(uuidv4 as () => string)
      .mockReturnValueOnce("new-rule")
      .mockReturnValueOnce("new-editor");
    const replacement = await createDraftForm();
    await act(async () => {
      await expect(
        applyMaskingRuleDraftInEditor(agentDraft, snapshot!, agentContext)
      ).rejects.toThrow();
    });
    expect(replacement).toHaveValue("Manual draft");
  });

  it("snapshots the current local fields and converted selector rather than the saved rule", async () => {
    const name = await createManualDraft();
    const before = await getMaskingRuleEditorSnapshot(agentContext);
    fireEvent.change(name, { target: { value: "Local phone rule" } });
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: "PARTIAL" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Change conditions" }));
    convertCondition.mockResolvedValueOnce(['resource.column_name == "phone"']);
    const snapshot = await getMaskingRuleEditorSnapshot(agentContext);
    expect(snapshot).toMatchObject({
      title: "Local phone rule",
      semanticType: "PARTIAL",
      condition: 'resource.column_name == "phone"',
      sessionId: before!.sessionId,
    });
    expect(snapshot!.revision).not.toBe(before!.revision);
    fireEvent.change(name, { target: { value: "x".repeat(201) } });
    expect(await getMaskingRuleEditorSnapshot(agentContext)).toBeUndefined();
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("reads existing rule edits for review without allowing full-rule draft application", async () => {
    const name = await editFirstRule();
    await waitFor(() => expect(name).toBeEnabled());
    expect(await getMaskingRuleEditorSnapshot(agentContext)).toMatchObject({
      mode: "edit",
      title: fixtures[0].condition.title,
      condition: fixtures[0].condition.expression,
    });
    await expect(
      applyMaskingRuleDraftInEditor(
        agentDraft,
        {
          sessionId: "unrelated",
          revision: "0",
          title: "Saved rule",
          condition: agentDraft.condition,
          semanticType: "DEFAULT",
        },
        agentContext
      )
    ).rejects.toThrow();
    expect(name).toHaveValue(fixtures[0].condition.title);
    expect(parseDraftCondition).not.toHaveBeenCalled();
    expect(
      screen.queryByRole("button", { name: "agent.masking-rule-draft.review" })
    ).not.toBeInTheDocument();
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("opens help without submitting, then reviews in the originating chat after selection", async () => {
    await renderPage();
    await act(async () => {
      await openMaskingRuleDraftInEditor(agentDraft, agentContext);
    });
    const request = openRuleHelp();
    expect(request.maskingRuleEditor).toBe(true);
    expect(agentState.queueUserMessage).not.toHaveBeenCalled();
    expect(openAgentPrompt).not.toHaveBeenCalled();
    act(() => expect(request.onStart?.("Review this rule.")).toBe(true));
    expect(openAgentPrompt).not.toHaveBeenCalled();
    expect(agentState.setCurrentChat).toHaveBeenCalledWith("chat-1");
    expect(agentState.queueUserMessage).toHaveBeenCalledWith({
      chatId: "chat-1",
      text: "Review this rule.\nagent.masking-rule-draft.help-prompt",
      runMode: "read-only",
      page: {
        path: "/settings/sensitive-data/global-masking",
        title: document.title,
      },
    });
    expect(agentState.open).toHaveBeenCalledOnce();
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue(
      agentDraft.title
    );
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("offers help for the current unsaved EDIT fields without saving or enabling full-rule apply", async () => {
    const name = await editFirstRule();
    await waitFor(() => expect(name).toBeEnabled());
    fireEvent.change(name, { target: { value: "Unsaved rule title" } });
    const request = openRuleHelp();
    expect(request.maskingRuleEditor).toBe(true);
    expect(openAgentPrompt).not.toHaveBeenCalled();
    expect(agentState.queueUserMessage).not.toHaveBeenCalled();
    act(() => expect(request.onStart?.("Explain this rule.")).toBe(true));
    const context = { ...agentContext, chatId: "manual-chat" };
    const snapshot = await getMaskingRuleEditorSnapshot(context);
    expect(snapshot).toMatchObject({ mode: "edit", title: "Unsaved rule title", chatId: "manual-chat" });
    await expect(applyMaskingRuleDraftInEditor(agentDraft, snapshot!, context)).rejects.toThrow();
    expect(name).toHaveValue("Unsaved rule title");
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("notifies open editor help when its pending condition parse finishes", async () => {
    await renderPage();
    let finishParsing!: (value: unknown[]) => void;
    parseCondition.mockImplementationOnce(() => new Promise((resolve) => {
      finishParsing = resolve;
    }));
    fireEvent.click(screen.getAllByRole("button", { name: prefix + "edit-rule" })[0]);
    const name = await screen.findByRole("textbox", { name: prefix + "name" });
    expect(name).toBeDisabled();
    const request = openRuleHelp();
    expect(request.canStart?.()).toBe(false);
    const readinessChanged = vi.fn(() => request.canStart?.());
    expect(request.subscribeCanStart).toBeTypeOf("function");
    const unsubscribe = request.subscribeCanStart?.(readinessChanged);
    try {
      await act(async () => finishParsing([{ exprKind: { case: "callExpr" } }]));
      expect(name).toBeEnabled();
      expect(readinessChanged).toHaveReturnedWith(true);
      expect(openAgentPrompt).not.toHaveBeenCalled();
      expect(agentState.queueUserMessage).not.toHaveBeenCalled();
    } finally {
      unsubscribe?.();
    }
  });

  it.each(["running", "awaiting_user", "queued", "archived", "missing"])(
    "keeps the toolbar available but blocks starter submission while the owning chat is %s",
    async (status) => {
      await renderPage();
      await act(async () => {
        await openMaskingRuleDraftInEditor(agentDraft, agentContext);
      });
      if (status === "missing") agentState.chats = [];
      else if (status === "archived") agentState.chats[0].archived = true;
      else if (status === "queued")
        agentState.queuedUserMessagesByChatId["chat-1"] = {};
      else agentState.chats[0].status = status;
      fireEvent.change(screen.getByRole("textbox", { name: prefix + "name" }), {
        target: { value: "Local draft" },
      });
      expect(screen.getByRole("button", { name: "agent.masking.ask-agent-short" })).toBeEnabled();
      const request = openRuleHelp();
      expect(request.canStart?.()).toBe(false);
      expect(request.onStart?.("Review this rule.")).toBe(false);
      expect(agentState.queueUserMessage).not.toHaveBeenCalled();
      expect(openAgentPrompt).not.toHaveBeenCalled();
      expect(agentState.open).not.toHaveBeenCalled();
    }
  );

  it("binds a manual draft only after a help starter is selected", async () => {
    await renderPage();
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "create-rule" })
    );
    const name = await screen.findByRole("textbox", { name: prefix + "name" });
    await waitFor(() => expect(name).toBeEnabled());
    fireEvent.change(name, { target: { value: "Manual draft" } });
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: "DEFAULT" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Change conditions" }));
    const request = openRuleHelp();
    expect(openAgentPrompt).not.toHaveBeenCalled();
    act(() => expect(request.onStart?.("Help me configure this rule.")).toBe(true));
    expect(openAgentPrompt).toHaveBeenCalledOnce();
    expect((await getMaskingRuleEditorSnapshot(agentContext))?.chatId).toBe(
      "manual-chat"
    );
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("reviews and refines a blank manual CREATE form without treating it as absent", async () => {
    await renderPage();
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "create-rule" })
    );
    const name = await screen.findByRole("textbox", { name: prefix + "name" });
    await waitFor(() => expect(name).toBeEnabled());
    validateCondition.mockReturnValue(false);
    const request = openRuleHelp();
    act(() => expect(request.onStart?.("Help me configure this rule.")).toBe(true));
    const reviewContext = { ...agentContext, chatId: "manual-chat" };
    const source = await getMaskingRuleEditorSnapshot(reviewContext);
    expect(source).toMatchObject({
      title: "",
      semanticType: "",
      condition: "",
      conditionStatus: "incomplete",
      chatId: "manual-chat",
    });
    validateCondition.mockReturnValue(true);
    await act(async () => {
      await applyMaskingRuleDraftInEditor(agentDraft, source!, reviewContext);
    });
    expect(name).toHaveValue(agentDraft.title);
    expect(screen.getByRole("combobox")).toHaveValue(agentDraft.semanticType);
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    expect(
      within(screen.getAllByRole("listitem").at(-1)!).getByRole("button", {
        name: prefix + "create-rule",
      })
    ).toBeEnabled();
  });

  it.each(["cancel", "actor"])("rejects help from a stale %s source", async (change) => {
    await createManualDraft();
    const request = openRuleHelp();
    expect(request.isAvailable?.()).toBe(true);
    if (change === "cancel") fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    else state.currentUser = { name: "users/another@example.com" };
    expect(request.isAvailable?.()).toBe(false);
    expect(request.onStart?.("Review this rule.")).toBe(false);
    if (change === "cancel") expect(revokePageHelp).toHaveBeenCalledOnce();
    expect(openAgentPrompt).not.toHaveBeenCalled();
    expect(agentState.queueUserMessage).not.toHaveBeenCalled();
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("preserves incomplete raw CEL as current draft text instead of saved policy", async () => {
    await createManualDraft();
    fireEvent.click(
      screen.getByRole("button", { name: "Incomplete raw condition" })
    );
    validateCondition.mockReturnValue(false);
    const snapshot = await getMaskingRuleEditorSnapshot(agentContext);
    expect(snapshot).toMatchObject({
      title: "Manual draft",
      condition: "resource.column_name == ",
      conditionStatus: "incomplete",
    });
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it("returns unavailable rather than throwing a snapshot conversion error", async () => {
    await createManualDraft();
    convertCondition.mockRejectedValueOnce(new Error("Conversion unavailable"));
    await expect(
      getMaskingRuleEditorSnapshot(agentContext)
    ).resolves.toBeUndefined();
  });

  it("discards a snapshot when its editor is replaced during CEL conversion", async () => {
    await createManualDraft();
    let complete!: (values: string[]) => void;
    convertCondition.mockImplementationOnce(() => new Promise((resolve) => { complete = resolve; }));
    const pending = getMaskingRuleEditorSnapshot(agentContext);
    await waitFor(() => expect(convertCondition).toHaveBeenCalled());
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    const replacement = await createDraftForm();
    fireEvent.change(replacement, { target: { value: "Replacement draft" } });
    await act(async () => complete([agentDraft.condition]));
    await expect(pending).resolves.toBeUndefined();
    expect(replacement).toHaveValue("Replacement draft");
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });

  it.each(["chat", "actor", "workspace"])(
    "discards a snapshot when %s identity changes during CEL conversion",
    async (change) => {
      await createManualDraft();
      let complete!: (values: string[]) => void;
      convertCondition.mockImplementationOnce(
        () =>
          new Promise((resolve) => {
            complete = resolve;
          })
      );
      const pending = getMaskingRuleEditorSnapshot(agentContext);
      await waitFor(() => expect(convertCondition).toHaveBeenCalled());
      if (change === "chat") agentState.currentChatId = "chat-2";
      else if (change === "actor")
        state.currentUser = { name: "users/other@example.com" };
      else agentState.chats[0].workspaceName = "workspaces/other";
      complete([agentDraft.condition]);
      expect(await pending).toBeUndefined();
    }
  );

  it.each([true, false])(
    "shows the original info banner with masking feature enabled: %s",
    async (enabled) => {
      state.hasInstanceFeature.mockReturnValue(enabled);
      await renderPage();
      const info = screen.getByRole("note");
      expect(info).toHaveTextContent(prefix + "description");
      if (enabled) {
        expect(info).toHaveTextContent("custom-approval.rule.first-match-wins");
      } else {
        expect(info).not.toHaveTextContent(
          "custom-approval.rule.first-match-wins"
        );
      }
      expect(
        within(info).getByRole("link", { name: "common.learn-more" })
      ).toHaveAttribute(
        "href",
        "https://docs.bytebase.com/security/data-masking/overview/?source=console"
      );
      expect(
        screen.queryByRole("heading", {
          name: "settings.sidebar.global-masking",
        })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole("button", { name: "common.learn-more" })
      ).not.toBeInTheDocument();
    }
  );
  it("renders ordered conditions with the read-only expression editor", async () => {
    await renderPage();
    expect(screen.getByText("Production masking")).toBeVisible();
    expect(screen.getByText("Default Partial")).toBeVisible();
    await waitFor(() => {
      for (const editor of screen.getAllByTestId("condition-editor")) {
        expect(editor).toHaveAttribute("data-readonly", "true");
        expect(editor).toHaveAttribute("data-bordered", "false");
        expect(editor.textContent).toBe(JSON.stringify(parsedExpr));
      }
    });
    expect(screen.queryByRole("combobox")).not.toBeInTheDocument();
    expect(screen.queryByText("Change conditions")).not.toBeInTheDocument();
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
  });
  it.each([
    "parse failure",
    "missing parsed expression",
    "empty parsed expression",
  ])("keeps raw CEL in the read-only editor on %s", async (failure) => {
    if (failure === "parse failure") {
      parseCondition.mockRejectedValue(new Error("Cannot parse expression"));
    } else if (failure === "empty parsed expression") {
      parseCondition.mockResolvedValue([{ exprKind: { case: undefined } }]);
    } else {
      parseCondition.mockResolvedValue([]);
    }
    await renderPage();
    await waitFor(() => {
      const editors = screen.getAllByTestId("condition-editor");
      expect(editors).toHaveLength(fixtures.length);
      editors.forEach((editor, index) => {
        expect(JSON.parse(editor.textContent!)).toEqual({
          type: "RawString",
          content: fixtures[index].condition.expression,
        });
        expect(editor).toHaveAttribute("data-readonly", "true");
      });
    });
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });
  it("stacks semantic type above condition details below the rule header", async () => {
    await renderPage();
    const rule = screen.getAllByRole("listitem")[0];
    const title = within(rule).getByText("Test masking");
    const edit = within(rule).getByRole("button", {
      name: prefix + "edit-rule",
    });
    const conditionLabel = within(rule).getByText(prefix + "condition");
    const semanticLabel = within(rule).getByText(prefix + "semantic-type");
    const condition = within(rule).getByTestId("condition-editor");
    const semanticType = within(rule).getByText("Default");
    const details = conditionLabel.closest("dl");
    expect(details).not.toBeNull();
    expect(semanticLabel.closest("dl")).toBe(details);
    expect(conditionLabel.tagName).toBe("DT");
    expect(semanticLabel.tagName).toBe("DT");
    expect(conditionLabel.nextElementSibling).toBe(condition.closest("dd"));
    expect(semanticLabel.nextElementSibling).toBe(semanticType.closest("dd"));
    expect(
      title.compareDocumentPosition(semanticLabel) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
    expect(
      semanticLabel.compareDocumentPosition(conditionLabel) &
        Node.DOCUMENT_POSITION_FOLLOWING
    ).toBeTruthy();
    expect(details).not.toContainElement(title);
    expect(title.parentElement).toContainElement(edit);
  });
  it("keeps semantic type descriptions out of the rule summary rows", async () => {
    await renderPage();
    expect(screen.getByText("Default")).toBeVisible();
    expect(screen.queryByText("Mask all values")).not.toBeInTheDocument();
    expect(
      screen.queryByText("Mask part of each value")
    ).not.toBeInTheDocument();
  });
  it("identifies the edit action on keyboard focus and opens its rule", async () => {
    await renderPage();
    const rule = screen.getAllByRole("listitem")[1];
    const edit = within(rule).getByRole("button", {
      name: prefix + "edit-rule",
    });
    fireEvent.focus(edit);
    expect(await screen.findByText(prefix + "edit-rule-tooltip")).toBeVisible();
    fireEvent.click(edit);
    expect(
      await screen.findByRole("textbox", { name: prefix + "name" })
    ).toHaveValue("Production masking");
    expect(screen.getByText("Test masking")).toBeVisible();
    expect(
      screen.getByRole("button", { name: prefix + "edit-rule" })
    ).toBeDisabled();
  });
  it("groups condition feedback with the expression field", async () => {
    validateCondition.mockReturnValue(false);
    await editFirstRule();
    const conditionField = await screen.findByRole("group", {
      name: prefix + "condition",
    });
    const error = within(conditionField).getByRole("alert");
    expect(error).toHaveAttribute("data-slot", "form-error");
    expect(error).toHaveTextContent(prefix + "error.invalid-condition");
    expect(conditionField).toHaveAttribute("aria-invalid", "true");
    expect(conditionField).toHaveAccessibleDescription(
      prefix + "error.invalid-condition"
    );
    const nameField = screen
      .getByRole("textbox", { name: prefix + "name" })
      .closest('[data-slot="form-field"]');
    const semanticField = screen
      .getByRole("combobox")
      .closest('[data-slot="form-field"]');
    expect(conditionField.parentElement).toBe(nameField?.parentElement);
    expect(semanticField?.parentElement).toBe(conditionField.parentElement);
    expect(nameField?.nextElementSibling).toBe(conditionField);
    expect(conditionField.nextElementSibling).toBe(semanticField);
  });
  it("constrains only the semantic selector while keeping the editor full width", async () => {
    const name = await editFirstRule();
    const selector = screen.getByRole("combobox", {
      name: prefix + "semantic-type",
    });
    expect(selector).toHaveClass("w-full", "max-w-sm");
    const fields = name.closest('[data-slot="form-field-group"]')!;
    expect(fields.parentElement).toHaveClass("w-full");
    expect(fields.parentElement).not.toHaveClass("max-w-sm");
    expect(fields.parentElement).not.toHaveClass("max-w-4xl");
    expect(name).not.toHaveClass("max-w-sm");
  });
  it.each([false, true])(
    "separates Delete rule from grouped Cancel and Save actions when dirty is %s",
    async (dirty) => {
      const name = await editFirstRule();
      if (dirty) fireEvent.change(name, { target: { value: "Unsaved title" } });
      const remove = screen.getByRole("button", {
        name: prefix + "delete-rule",
      });
      const cancel = screen.getByRole("button", { name: "common.cancel" });
      const save = screen.getByRole("button", { name: prefix + "save-rule" });
      expect(remove).toHaveTextContent(prefix + "delete-rule");
      expect(cancel.parentElement).toBe(save.parentElement);
      expect(cancel.parentElement).not.toContainElement(remove);
      expect(
        name.closest('[data-slot="form-field-group"]')
      ).not.toContainElement(remove);
      expect(
        remove.compareDocumentPosition(cancel) &
          Node.DOCUMENT_POSITION_FOLLOWING
      ).toBeTruthy();
      if (dirty) {
        const footer = save.closest('[data-slot="sticky-action-footer"]');
        expect(footer).toContainElement(remove);
        expect(footer).toContainElement(cancel);
        expect(
          remove.closest('[data-slot="sticky-action-footer-left"]')
        ).not.toBeNull();
        expect(
          cancel.closest('[data-slot="sticky-action-footer-right"]')
        ).not.toBeNull();
      } else {
        expect(save.closest('[data-slot="sticky-action-footer"]')).toBeNull();
        expect(cancel.parentElement?.parentElement).toContainElement(remove);
      }
    }
  );
  it("uses external field titles and delegates the editor frame", async () => {
    await editFirstRule();
    const conditionField = screen.getByRole("group", {
      name: prefix + "condition",
    });
    const editor = within(conditionField).getByTestId("condition-editor");
    expect(editor).toHaveAttribute("data-bordered", "true");
    expect(editor.parentElement).toBe(conditionField);
    expect(
      within(conditionField).getByText(prefix + "condition")
    ).toBeVisible();
    expect(
      screen.getByRole("combobox", { name: prefix + "semantic-type" })
    ).toBeVisible();
  });
  it("connects missing semantic type feedback to the selector", async () => {
    await renderPage();
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "create-rule" })
    );
    const selector = await screen.findByRole("combobox");
    await waitFor(() => expect(selector).toBeEnabled());
    const semanticField = selector.closest('[data-slot="form-field"]')!;
    expect(
      within(semanticField as HTMLElement).getByRole("alert")
    ).toHaveTextContent(prefix + "error.missing-semantic-type");
    expect(selector).toHaveAttribute("aria-invalid", "true");
    expect(selector).toHaveAccessibleDescription(
      prefix + "error.missing-semantic-type"
    );
    fireEvent.change(selector, { target: { value: "PARTIAL" } });
    expect(
      within(semanticField as HTMLElement).queryByRole("alert")
    ).not.toBeInTheDocument();
    expect(selector).not.toHaveAttribute("aria-describedby");
  });
  it("allows one editor and cancels without changing the saved rule", async () => {
    const name = await editFirstRule();
    fireEvent.change(name, { target: { value: "Unsaved title" } });
    expect(
      screen.getByRole("button", { name: prefix + "create-rule" })
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: prefix + "re-order" })
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: prefix + "edit-rule" })
    ).toBeDisabled();
    await waitFor(() => expect(guard).toHaveBeenLastCalledWith(true));
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(await screen.findByText("Test masking")).toBeVisible();
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    expect(guard).toHaveBeenLastCalledWith(false);
  });
  it("saves only the edited rule and preserves metadata and other rules", async () => {
    const name = await editFirstRule();
    expect(
      screen.getByRole("button", { name: prefix + "save-rule" })
    ).toBeDisabled();
    fireEvent.change(name, { target: { value: "Updated rule" } });
    fireEvent.click(screen.getByRole("button", { name: prefix + "save-rule" }));
    await screen.findByText("Updated rule");
    expect(savedRules()).toEqual([
      {
        ...fixtures[0],
        condition: { ...fixtures[0].condition, title: "Updated rule" },
      },
      fixtures[1],
    ]);
  });
  it("retains a failed save draft and permits retry", async () => {
    state.upsertPolicy.mockRejectedValueOnce(new Error("write failed"));
    const name = await editFirstRule();
    fireEvent.change(name, { target: { value: "Retry title" } });
    fireEvent.click(screen.getByRole("button", { name: prefix + "save-rule" }));
    await screen.findByRole("alert");
    expect(name).toHaveValue("Retry title");
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toBe(name);
    fireEvent.click(screen.getByRole("button", { name: prefix + "save-rule" }));
    await screen.findByText("Retry title");
    expect(state.upsertPolicy).toHaveBeenCalledTimes(2);
  });
  it("locks a pending save without closing the editor optimistically", async () => {
    let resolveWrite!: () => void;
    state.upsertPolicy.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveWrite = () => resolve(undefined);
        })
    );
    const name = await editFirstRule();
    fireEvent.change(name, { target: { value: "Pending rule" } });
    fireEvent.click(screen.getByRole("button", { name: prefix + "save-rule" }));
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(1));
    expect(name).toBeDisabled();
    expect(
      screen.getByRole("button", { name: "common.cancel" })
    ).toBeDisabled();
    expect(
      screen.getByRole("button", { name: prefix + "delete-rule" })
    ).toBeDisabled();
    await act(async () => resolveWrite());
    await screen.findByText("Pending rule");
  });
  it("discards a new rule without writing", async () => {
    await renderPage();
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "create-rule" })
    );
    await screen.findByRole("textbox", { name: prefix + "name" });
    expect(
      screen.queryByRole("button", { name: prefix + "delete-rule" })
    ).not.toBeInTheDocument();
    expect(
      screen
        .getAllByRole("button", { name: prefix + "create-rule" })
        .every((button) => button.hasAttribute("disabled"))
    ).toBe(true);
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });
  it("restores canceled order without refetching or writing", async () => {
    await renderPage();
    fireEvent.click(screen.getByRole("button", { name: prefix + "re-order" }));
    fireEvent.click(
      screen.getAllByRole("button", { name: prefix + "move-down" })[0]
    );
    expect(
      within(screen.getAllByRole("listitem")[0]).getByText("Production masking")
    ).toBeVisible();
    fireEvent.click(screen.getByRole("button", { name: "common.cancel" }));
    expect(
      within(screen.getAllByRole("listitem")[0]).getByText("Test masking")
    ).toBeVisible();
    expect(state.getOrFetchPolicyByParentAndType).toHaveBeenCalledTimes(1);
    expect(state.upsertPolicy).not.toHaveBeenCalled();
  });
  it("writes reordered rules only on Save order", async () => {
    await renderPage();
    fireEvent.click(screen.getByRole("button", { name: prefix + "re-order" }));
    expect(
      screen.getByRole("button", { name: prefix + "save-order" })
    ).toBeDisabled();
    fireEvent.click(
      screen.getAllByRole("button", { name: prefix + "move-down" })[0]
    );
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "save-order" })
    );
    await waitFor(() =>
      expect(savedRules()).toEqual([fixtures[1], fixtures[0]])
    );
    await screen.findByRole("button", { name: prefix + "re-order" });
  });
  it("retains a failed reorder draft", async () => {
    state.upsertPolicy.mockRejectedValueOnce(new Error("write failed"));
    await renderPage();
    fireEvent.click(screen.getByRole("button", { name: prefix + "re-order" }));
    fireEvent.click(
      screen.getAllByRole("button", { name: prefix + "move-down" })[0]
    );
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "save-order" })
    );
    await screen.findByRole("alert");
    expect(
      within(screen.getAllByRole("listitem")[0]).getByText("Production masking")
    ).toBeVisible();
    expect(
      screen.getByRole("button", { name: prefix + "save-order" })
    ).toBeEnabled();
  });
  it("uses a confirmation popover and preserves other rules on deletion", async () => {
    await editFirstRule();
    const trigger = screen.getByRole("button", {
      name: prefix + "delete-rule",
    });
    fireEvent.click(trigger);
    const popup = await screen.findByRole("dialog", { name: "common.delete" });
    expect(trigger).toHaveAttribute("aria-expanded", "true");
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    fireEvent.click(
      within(popup).getByRole("button", { name: "common.cancel" })
    );
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    fireEvent.click(trigger);
    fireEvent.click(
      within(await screen.findByRole("dialog")).getByRole("button", {
        name: "common.delete",
      })
    );
    await waitFor(() => expect(savedRules()).toEqual([fixtures[1]]));
    await waitFor(() =>
      expect(screen.queryByRole("textbox")).not.toBeInTheDocument()
    );
  });

  it("creates a rule only after valid draft fields are confirmed", async () => {
    await renderPage();
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "create-rule" })
    );
    const name = await screen.findByRole("textbox", { name: prefix + "name" });
    await waitFor(() => expect(name).toBeEnabled());
    fireEvent.change(name, { target: { value: "New masking rule" } });
    fireEvent.change(screen.getByRole("combobox"), {
      target: { value: "PARTIAL" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Change conditions" }));
    expect(state.upsertPolicy).not.toHaveBeenCalled();
    const editor = screen.getAllByRole("listitem").at(-1)!;
    fireEvent.click(
      within(editor).getByRole("button", { name: prefix + "create-rule" })
    );
    await screen.findByText("New masking rule");
    expect(savedRules().slice(0, 2)).toEqual(fixtures);
    expect(savedRules()[2]).toMatchObject({
      id: "new-rule",
      semanticType: "PARTIAL",
      condition: {
        title: "New masking rule",
        expression: fixtures[0].condition.expression,
      },
    });
    expect(state.upsertPolicy).toHaveBeenCalledTimes(1);
  });

  it("retains a rule after failed deletion and permits retry", async () => {
    state.upsertPolicy.mockRejectedValueOnce(new Error("delete failed"));
    await editFirstRule();
    fireEvent.click(
      screen.getByRole("button", { name: prefix + "delete-rule" })
    );
    const popup = await screen.findByRole("dialog");
    fireEvent.click(
      within(popup).getByRole("button", { name: "common.delete" })
    );
    await screen.findByRole("alert");
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(screen.getByRole("textbox", { name: prefix + "name" })).toHaveValue(
      "Test masking"
    );
    fireEvent.click(
      within(popup).getByRole("button", { name: "common.delete" })
    );
    await waitFor(() => expect(savedRules()).toEqual([fixtures[1]]));
    await waitFor(() =>
      expect(screen.queryByRole("textbox")).not.toBeInTheDocument()
    );
    expect(state.upsertPolicy).toHaveBeenCalledTimes(2);
  });

  it("locks the editor and confirmation actions while deletion is pending", async () => {
    let resolveWrite!: () => void;
    state.upsertPolicy.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          resolveWrite = () => resolve(undefined);
        })
    );
    const name = await editFirstRule();
    const trigger = screen.getByRole("button", {
      name: prefix + "delete-rule",
    });
    fireEvent.click(trigger);
    const popup = await screen.findByRole("dialog", { name: "common.delete" });
    const confirm = within(popup).getByRole("button", {
      name: "common.delete",
    });
    fireEvent.click(confirm);
    await waitFor(() => expect(state.upsertPolicy).toHaveBeenCalledTimes(1));
    expect(name).toBeDisabled();
    expect(trigger).toBeDisabled();
    expect(confirm).toBeDisabled();
    for (const cancel of screen.getAllByRole("button", {
      name: "common.cancel",
    })) {
      expect(cancel).toBeDisabled();
    }
    expect(screen.getAllByRole("listitem")).toHaveLength(2);
    expect(popup).toBeVisible();
    await act(async () => resolveWrite());
    await waitFor(() =>
      expect(screen.queryByRole("dialog")).not.toBeInTheDocument()
    );
    expect(screen.getAllByRole("listitem")).toHaveLength(1);
  });

  it("disables saving again when an edit is reverted to the saved value", async () => {
    const name = await editFirstRule();
    fireEvent.change(name, { target: { value: "Temporary title" } });
    expect(
      screen.getByRole("button", { name: prefix + "save-rule" })
    ).toBeEnabled();
    fireEvent.change(name, { target: { value: "Test masking" } });
    expect(
      screen.getByRole("button", { name: prefix + "save-rule" })
    ).toBeDisabled();
    await waitFor(() => expect(guard).toHaveBeenLastCalledWith(false));
  });
  it.each(["permission", "feature"])(
    "blocks writes without the %s",
    async (limitation) => {
      if (limitation === "permission") state.hasPermission = false;
      else state.hasInstanceFeature.mockReturnValue(false);
      await renderPage();
      expect(
        screen.getByRole("button", { name: prefix + "create-rule" })
      ).toBeDisabled();
      expect(
        screen.getByRole("button", { name: prefix + "re-order" })
      ).toBeDisabled();
      expect(
        screen.queryByRole("button", { name: prefix + "edit-rule" })
      ).not.toBeInTheDocument();
      expect(state.upsertPolicy).not.toHaveBeenCalled();
    }
  );
});
