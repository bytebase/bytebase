import { create } from "@bufbuild/protobuf";
import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import { State } from "@/types/proto-es/v1/common_pb";
import {
  BindingSchema,
  type IamPolicy,
  IamPolicySchema,
} from "@/types/proto-es/v1/iam_policy_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import { type User, UserSchema } from "@/types/proto-es/v1/user_service_pb";
import {
  type AppStoreUtilBridge,
  registerAppStoreUtilBridge,
} from "@/utils/app-store-bridge";
import { useApprovalCandidates } from "./useApprovalCandidates";

const ROLE = "roles/servicenow-approver";
const PROJECT = "projects/orders";
const WORKSPACE_SA = "integration@service.bytebase.com";
const PROJECT_SA = "testdt@orders.service.bytebase.com";
const WORKLOAD_IDENTITY = "deploy@orders.workload.bytebase.com";

type Account = { name: string; email: string; title: string; state: State };

const store = vi.hoisted(() => ({
  projectPoliciesByName: {} as Record<string, unknown>,
  groupsByName: {},
  batchGetOrFetchGroups: vi.fn(async () => []),
  batchGetOrFetchUsers: vi.fn(),
  fetchServiceAccount: vi.fn(),
  fetchWorkloadIdentity: vi.fn(),
}));
const session = vi.hoisted(() => ({
  currentUserEmail: "",
  project: { allowSelfApproval: false, allowLastPlanEditorApproval: true },
}));

vi.mock("@/stores/app", () => ({
  useAppStore: (selector: (state: unknown) => unknown) => selector(store),
}));
vi.mock("@/hooks/useAppState", () => ({
  useCurrentUser: () => ({ email: session.currentUserEmail }),
}));
vi.mock("@/hooks/useProjectByName", () => ({
  useProjectByName: () => session.project,
}));

const person = (email: string, title: string, state = State.ACTIVE): User =>
  create(UserSchema, { name: `users/${email}`, email, title, state });

const account = (email: string, title: string, state = State.ACTIVE) => ({
  email,
  title,
  state,
});

const policy = (members: string[]): IamPolicy =>
  create(IamPolicySchema, {
    bindings: [create(BindingSchema, { role: ROLE, members })],
  });

// What each service knows; a name missing here does not resolve.
let people: Record<string, User>;
let serviceAccounts: Record<string, Omit<Account, "name">>;
let workloadIdentities: Record<string, Omit<Account, "name">>;
let workspaceHolders: string[];

const issue = (creator = "users/creator@example.com") =>
  ({ creator }) as unknown as Issue;

const candidatesOf = async (issueArg = issue(), role = ROLE) => {
  const { result } = renderHook(() =>
    useApprovalCandidates(issueArg, "orders", role, "")
  );
  return result;
};

beforeEach(() => {
  people = {};
  serviceAccounts = {};
  workloadIdentities = {};
  workspaceHolders = [];
  session.currentUserEmail = "";
  session.project = {
    allowSelfApproval: false,
    allowLastPlanEditorApproval: true,
  };
  store.projectPoliciesByName = {};
  store.batchGetOrFetchUsers.mockReset();
  // As the store answers: a name it cannot resolve comes back as a placeholder
  // named from its email.
  store.batchGetOrFetchUsers.mockImplementation(async (names: string[]) =>
    names.map(
      (name) =>
        people[name] ??
        person(name.slice("users/".length), name.slice(6).split("@")[0])
    )
  );
  store.fetchServiceAccount.mockReset();
  store.fetchServiceAccount.mockImplementation(
    async (name: string) => serviceAccounts[name]
  );
  store.fetchWorkloadIdentity.mockReset();
  store.fetchWorkloadIdentity.mockImplementation(
    async (name: string) => workloadIdentities[name]
  );
  registerAppStoreUtilBridge({
    getGroupByIdentifier: () => undefined,
    workspaceRoleMapToUsers: () => new Map([[ROLE, new Set(workspaceHolders)]]),
  } as unknown as AppStoreUtilBridge);
});

describe("useApprovalCandidates", () => {
  test("a role held only by service accounts and a workload identity lists each of them", async () => {
    workspaceHolders = [`serviceAccounts/${WORKSPACE_SA}`];
    store.projectPoliciesByName[PROJECT] = policy([
      `serviceAccount:${PROJECT_SA}`,
      `workloadIdentity:${WORKLOAD_IDENTITY}`,
    ]);
    serviceAccounts[`serviceAccounts/${WORKSPACE_SA}`] = account(
      WORKSPACE_SA,
      "ServiceNow integration"
    );
    serviceAccounts[`serviceAccounts/${PROJECT_SA}`] = account(
      PROJECT_SA,
      "testdt"
    );
    workloadIdentities[`workloadIdentities/${WORKLOAD_IDENTITY}`] = account(
      WORKLOAD_IDENTITY,
      "Deploy pipeline"
    );

    const result = await candidatesOf();

    await waitFor(() => expect(result.current.candidates).toHaveLength(3));
    expect(
      result.current.candidates.map(({ user }) => [user.title, user.email])
    ).toEqual([
      ["Deploy pipeline", WORKLOAD_IDENTITY],
      ["ServiceNow integration", WORKSPACE_SA],
      ["testdt", PROJECT_SA],
    ]);
    expect(store.batchGetOrFetchUsers).not.toHaveBeenCalledWith(
      expect.arrayContaining([expect.stringContaining("service.bytebase.com")])
    );
  });

  test("people come before service accounts, and the signed-in reviewer first", async () => {
    session.currentUserEmail = "zoe@example.com";
    store.projectPoliciesByName[PROJECT] = policy([
      `serviceAccount:${PROJECT_SA}`,
      "user:zoe@example.com",
      "user:adam@example.com",
    ]);
    people["users/zoe@example.com"] = person("zoe@example.com", "Zoe");
    people["users/adam@example.com"] = person("adam@example.com", "Adam");
    serviceAccounts[`serviceAccounts/${PROJECT_SA}`] = account(PROJECT_SA, "A");

    const result = await candidatesOf();

    await waitFor(() => expect(result.current.candidates).toHaveLength(3));
    expect(result.current.candidates.map(({ user }) => user.title)).toEqual([
      "Zoe",
      "Adam",
      "A",
    ]);
    expect(result.current.isCurrentUserRoleMember).toBe(true);
  });

  test("nobody holds the role: no candidates", async () => {
    store.projectPoliciesByName[PROJECT] = policy([]);

    const result = await candidatesOf();

    await waitFor(() =>
      expect(store.batchGetOrFetchGroups).toHaveBeenCalled()
    );
    expect(result.current.candidates).toEqual([]);
  });

  test("a deleted service account is not a candidate; one whose record does not resolve still is", async () => {
    store.projectPoliciesByName[PROJECT] = policy([
      `serviceAccount:${PROJECT_SA}`,
      `serviceAccount:${WORKSPACE_SA}`,
    ]);
    serviceAccounts[`serviceAccounts/${PROJECT_SA}`] = account(
      PROJECT_SA,
      "testdt",
      State.DELETED
    );

    const result = await candidatesOf();

    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    expect(result.current.candidates[0].user).toMatchObject({
      email: WORKSPACE_SA,
      title: "integration",
    });
  });

  test("a service account that created the issue is held to the self-approval rule", async () => {
    store.projectPoliciesByName[PROJECT] = policy([
      `serviceAccount:${WORKSPACE_SA}`,
    ]);
    serviceAccounts[`serviceAccounts/${WORKSPACE_SA}`] = account(
      WORKSPACE_SA,
      "ServiceNow integration"
    );

    const result = await candidatesOf(issue(`users/${WORKSPACE_SA}`));

    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    expect(result.current.candidates[0]).toMatchObject({
      canApprove: false,
      ineligibilities: ["self-approval"],
    });
  });

  test("a service account bound under user: holds nothing and is not listed", async () => {
    store.projectPoliciesByName[PROJECT] = policy([
      `user:${WORKSPACE_SA}`,
      "user:adam@example.com",
    ]);
    people["users/adam@example.com"] = person("adam@example.com", "Adam");

    const result = await candidatesOf();

    await waitFor(() => expect(result.current.candidates).toHaveLength(1));
    expect(result.current.candidates[0].user.title).toBe("Adam");
  });
});
