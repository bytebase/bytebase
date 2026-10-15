import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { type ReactNode, useState } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Policy,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewAIPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";

const SWITCH = { name: "sql-review.ai-review.enable.self" };
const POLICY_EDITOR = { name: "sql-review.ai-review.policy.self" };
const WORKSPACE = "workspaces/ws";

const mocks = vi.hoisted(() => ({
  permissions: {} as Record<string, boolean>,
  saas: { value: false },
  upsertSetting: vi.fn(),
  upsertPolicy: vi.fn(),
  pushNotification: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}));

vi.mock("@/hooks/useAppState", () => ({
  useServerState: () => ({ isSaaSMode: mocks.saas.value }),
  useWorkspaceResourceName: () => "workspaces/ws",
}));

vi.mock("@/utils", () => ({
  hasWorkspacePermissionV2: (permission: string) =>
    mocks.permissions[permission] ?? true,
}));

vi.mock("@/stores", () => ({ pushNotification: mocks.pushNotification }));

vi.mock("@/components/RouterLink", () => ({
  RouterLink: ({ children }: { children: ReactNode }) => (
    <a href="/">{children}</a>
  ),
}));

// Mirrors the real store: a saved AI setting updates the server info, a fetch
// of a missing policy caches one with no payload and resolves to null, and a
// saved policy is cached under its name.
vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  const { create: createMessage } = await import("@bufbuild/protobuf");
  const { PolicySchema: Schema, PolicyType: Type } = await import(
    "@/types/proto-es/v1/org_policy_service_pb"
  );
  type StorePolicy = import("@/types/proto-es/v1/org_policy_service_pb").Policy;
  type Find = { parentPath: string; policyType: number };
  const nameOf = (parentPath: string, policyType: number) =>
    `${parentPath}/policies/${Type[policyType].toLowerCase()}`;
  type ServerInfo = { aiEnabled: boolean; aiSqlReviewEnabled: boolean };
  type Saved = {
    value: { value: { value: { sqlReviewEnabled: boolean } } };
  };
  const useAppStore = createStore<{
    serverInfo: ServerInfo;
    policies: Record<string, StorePolicy>;
    upsertSetting: (params: Saved) => Promise<void>;
    getPolicyByParentAndType: (find: Find) => StorePolicy | undefined;
    fetchPolicyByParentAndType: (find: Find) => Promise<StorePolicy | null>;
    upsertPolicy: (params: {
      parentPath: string;
      policy: Partial<StorePolicy>;
    }) => Promise<StorePolicy>;
  }>()((set, get) => ({
    serverInfo: { aiEnabled: true, aiSqlReviewEnabled: false },
    policies: {},
    upsertSetting: async (params) => {
      await mocks.upsertSetting(params);
      set((state) => ({
        serverInfo: {
          ...state.serverInfo,
          aiSqlReviewEnabled: params.value.value.value.sqlReviewEnabled,
        },
      }));
    },
    getPolicyByParentAndType: (find) =>
      get().policies[nameOf(find.parentPath, find.policyType)],
    fetchPolicyByParentAndType: async (find) => {
      const name = nameOf(find.parentPath, find.policyType);
      if (!get().policies[name]) {
        set((state) => ({
          policies: {
            ...state.policies,
            [name]: createMessage(Schema, { name }),
          },
        }));
        return null;
      }
      return get().policies[name];
    },
    upsertPolicy: async (params) => {
      await mocks.upsertPolicy(params);
      const name = nameOf(params.parentPath, params.policy.type!);
      const saved = { ...params.policy, name } as StorePolicy;
      set((state) => ({ policies: { ...state.policies, [name]: saved } }));
      return saved;
    },
  }));
  return { useAppStore };
});

const { useAppStore } = await import("@/stores/app");
const seedServerInfo = (serverInfo: {
  aiEnabled: boolean;
  aiSqlReviewEnabled: boolean;
}) =>
  (
    useAppStore as unknown as {
      setState: (state: { serverInfo: typeof serverInfo }) => void;
    }
  ).setState({ serverInfo });
const seedPolicies = (policies: Record<string, Policy>) =>
  (
    useAppStore as unknown as {
      setState: (state: { policies: Record<string, Policy> }) => void;
    }
  ).setState({ policies });

const workspacePolicy = (content: string) =>
  create(PolicySchema, {
    name: `${WORKSPACE}/policies/review_ai`,
    type: PolicyType.REVIEW_AI,
    enforce: true,
    policy: {
      case: "reviewAiPolicy",
      value: create(ReviewAIPolicySchema, { content }),
    },
  });

const {
  SQLReviewAIReviewSection,
  useWorkspaceAIReview,
  useWorkspaceReviewAIPolicy,
} = await import("./SQLReviewAIReviewSection");

function Harness() {
  const aiReview = useWorkspaceAIReview();
  const policy = useWorkspaceReviewAIPolicy(true);
  const [saved, setSaved] = useState<boolean>();
  const [policySaved, setPolicySaved] = useState<boolean>();
  return (
    <>
      <SQLReviewAIReviewSection aiReview={aiReview} policy={policy} />
      <output>{aiReview.isDirty ? "dirty" : "clean"}</output>
      <output>{`saved: ${saved}`}</output>
      <output>{`policy ${policy.isDirty ? "dirty" : "clean"}`}</output>
      <output>{`policy saved: ${policySaved}`}</output>
      <button
        type="button"
        onClick={() => void aiReview.save().then(setSaved)}
      >
        save
      </button>
      <button type="button" onClick={aiReview.revert}>
        revert
      </button>
      <button
        type="button"
        onClick={() => void policy.save().then(setPolicySaved)}
      >
        save policy
      </button>
    </>
  );
}

const save = () =>
  act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "save" }));
  });

describe("SQLReviewAIReviewSection", () => {
  beforeEach(() => {
    mocks.permissions = {};
    mocks.saas.value = false;
    mocks.upsertSetting.mockReset();
    mocks.upsertPolicy.mockReset();
    mocks.pushNotification.mockReset();
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: false });
    seedPolicies({});
  });

  test("loads the workspace policy, and saves an edit under the workspace", async () => {
    seedPolicies({
      [`${WORKSPACE}/policies/review_ai`]: workspacePolicy("No DROP."),
    });
    render(<Harness />);

    const editor = await screen.findByRole("textbox", POLICY_EDITOR);
    expect(editor).toHaveValue("No DROP.");
    expect(editor).toHaveAccessibleDescription(
      "sql-review.ai-review.policy.description"
    );
    fireEvent.change(editor, { target: { value: "No DROP or TRUNCATE." } });
    expect(screen.getByText("policy dirty")).toBeInTheDocument();
    expect(screen.getByText("clean")).toBeInTheDocument();
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "save policy" }));
    });

    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    const { parentPath, policy } = mocks.upsertPolicy.mock.calls[0][0];
    expect(parentPath).toBe(WORKSPACE);
    expect(policy.type).toBe(PolicyType.REVIEW_AI);
    expect(policy.resourceType).toBe(PolicyResourceType.WORKSPACE);
    expect(policy.policy.case).toBe("reviewAiPolicy");
    expect(policy.policy.value.content).toBe("No DROP or TRUNCATE.");
    expect(screen.getByText("policy clean")).toBeInTheDocument();
    expect(screen.getByText("policy saved: true")).toBeInTheDocument();
    expect(mocks.upsertSetting).not.toHaveBeenCalled();
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("a failed policy save keeps the edit", async () => {
    mocks.upsertPolicy.mockRejectedValueOnce(new Error("denied"));
    render(<Harness />);

    const editor = await screen.findByRole("textbox", POLICY_EDITOR);
    fireEvent.change(editor, { target: { value: "No DROP." } });
    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "save policy" }));
    });

    expect(editor).toHaveValue("No DROP.");
    expect(screen.getByText("policy dirty")).toBeInTheDocument();
    expect(screen.getByText("policy saved: false")).toBeInTheDocument();
  });

  test("shows the workspace policy as text without bb.policies.update or bb.policies.delete", async () => {
    mocks.permissions["bb.policies.update"] = false;
    mocks.permissions["bb.policies.delete"] = false;
    seedPolicies({
      [`${WORKSPACE}/policies/review_ai`]: workspacePolicy("No DROP."),
    });
    render(<Harness />);

    expect(await screen.findByText("No DROP.")).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  test("saves the switch under its own mask path", async () => {
    render(<Harness />);
    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "false"
    );
    expect(screen.getByText("clean")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("switch", SWITCH));
    expect(screen.getByText("dirty")).toBeInTheDocument();
    await save();

    expect(mocks.upsertSetting).toHaveBeenCalledOnce();
    const saved = mocks.upsertSetting.mock.calls[0][0];
    expect(saved.name).toBe(Setting_SettingName.AI);
    expect(saved.updateMask.paths).toEqual(["value.ai.sql_review_enabled"]);
    expect(saved.value.value.value.sqlReviewEnabled).toBe(true);
    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByText("clean")).toBeInTheDocument();
    expect(screen.getByText("saved: true")).toBeInTheDocument();
    // The page reports the update, once for everything the click saved.
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("reverting drops the unsaved choice", () => {
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: true });
    render(<Harness />);

    fireEvent.click(screen.getByRole("switch", SWITCH));
    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "false"
    );
    fireEvent.click(screen.getByRole("button", { name: "revert" }));

    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByText("clean")).toBeInTheDocument();
  });

  test("a failed save keeps the unsaved choice", async () => {
    mocks.upsertSetting.mockRejectedValueOnce(new Error("save failed"));
    render(<Harness />);

    fireEvent.click(screen.getByRole("switch", SWITCH));
    await save();

    expect(screen.getByRole("switch", SWITCH)).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByText("dirty")).toBeInTheDocument();
    expect(screen.getByText("saved: false")).toBeInTheDocument();
    expect(mocks.pushNotification).not.toHaveBeenCalled();
  });

  test("shows the state as text without bb.settings.set", () => {
    mocks.permissions["bb.settings.set"] = false;
    seedServerInfo({ aiEnabled: true, aiSqlReviewEnabled: true });
    render(<Harness />);

    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(screen.getByText("sql-review.standard-rules.on")).toBeInTheDocument();
  });

  test("shows the state as text in SaaS mode, where the AI setting is read only", () => {
    mocks.saas.value = true;
    render(<Harness />);

    expect(screen.queryByRole("switch")).not.toBeInTheDocument();
    expect(
      screen.getByText("sql-review.standard-rules.off")
    ).toBeInTheDocument();
  });

  test("warns while AI is off for the workspace, with a link to turn it on", () => {
    seedServerInfo({ aiEnabled: false, aiSqlReviewEnabled: true });
    render(<Harness />);

    expect(
      screen.getByText("sql-review.ai-review.ai-off")
    ).toBeInTheDocument();
    expect(
      screen.getByRole("link", { name: "sql-review.ai-review.turn-on-ai" })
    ).toBeInTheDocument();
  });

  test.each([
    ["without bb.settings.set", () => (mocks.permissions["bb.settings.set"] = false)],
    ["without bb.settings.get", () => (mocks.permissions["bb.settings.get"] = false)],
    [
      "without bb.settings.getWorkspaceProfile",
      () => (mocks.permissions["bb.settings.getWorkspaceProfile"] = false),
    ],
    ["in SaaS mode", () => (mocks.saas.value = true)],
  ])("warns without the link %s, where the user cannot turn AI on", (_name, arrange) => {
    arrange();
    seedServerInfo({ aiEnabled: false, aiSqlReviewEnabled: false });
    render(<Harness />);

    expect(
      screen.getByText("sql-review.ai-review.ai-off")
    ).toBeInTheDocument();
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });

  test("shows no warning while AI is on", () => {
    render(<Harness />);

    expect(
      screen.queryByText("sql-review.ai-review.ai-off")
    ).not.toBeInTheDocument();
  });
});
