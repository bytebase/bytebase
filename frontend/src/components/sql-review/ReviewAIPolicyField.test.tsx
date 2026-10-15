import { create } from "@bufbuild/protobuf";
import { act, fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { beforeEach, describe, expect, test, vi } from "vitest";
import {
  type Policy,
  PolicyResourceType,
  PolicySchema,
  PolicyType,
  ReviewAIPolicySchema,
} from "@/types/proto-es/v1/org_policy_service_pb";

const PARENT = "projects/p";
const POLICY_NAME = `${PARENT}/policies/review_ai`;
const EDITOR = { name: "sql-review.ai-review.policy.self" };

const mocks = vi.hoisted(() => ({
  permissions: {} as Record<string, boolean>,
  // Policy names whose read fails: the store then caches nothing for them.
  unreadable: new Set<string>(),
  fetchPolicy: vi.fn(),
  upsertPolicy: vi.fn(),
  deletePolicy: vi.fn(),
}));

vi.mock("react-i18next", () => ({
  useTranslation: () => ({
    t: (key: string, params?: Record<string, string>) =>
      params ? `${key} ${JSON.stringify(params)}` : key,
  }),
}));

// Mirrors the real store: a fetch of a missing policy caches one with no
// payload and resolves to null, a failed read resolves to undefined and
// leaves the cache as it was, an update keeps the stored enforce flag, a
// create is enforced, and a delete drops the cached entry.
vi.mock("@/stores/app", async () => {
  const { create: createStore } = await import("zustand");
  const { create: createMessage } = await import("@bufbuild/protobuf");
  const { PolicySchema: Schema, PolicyType: Type } = await import(
    "@/types/proto-es/v1/org_policy_service_pb"
  );
  type StorePolicy = import("@/types/proto-es/v1/org_policy_service_pb").Policy;
  type Find = { parentPath: string; policyType: number; refresh?: boolean };
  const nameOf = (parentPath: string, policyType: number) =>
    `${parentPath}/policies/${Type[policyType].toLowerCase()}`;
  const useAppStore = createStore<{
    policies: Record<string, StorePolicy>;
    getPolicyByParentAndType: (find: Find) => StorePolicy | undefined;
    fetchPolicyByParentAndType: (
      find: Find
    ) => Promise<StorePolicy | null | undefined>;
    upsertPolicy: (params: {
      parentPath: string;
      policy: Partial<StorePolicy>;
    }) => Promise<StorePolicy>;
    deletePolicy: (name: string) => Promise<void>;
  }>()((set, get) => ({
    policies: {},
    getPolicyByParentAndType: (find) =>
      get().policies[nameOf(find.parentPath, find.policyType)],
    fetchPolicyByParentAndType: async (find) => {
      await mocks.fetchPolicy(find);
      const name = nameOf(find.parentPath, find.policyType);
      if (mocks.unreadable.has(name)) return undefined;
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
      const existing = get().policies[name];
      const saved = {
        ...params.policy,
        name,
        enforce: existing?.policy.case ? existing.enforce : true,
      } as StorePolicy;
      set((state) => ({ policies: { ...state.policies, [name]: saved } }));
      return saved;
    },
    deletePolicy: async (name) => {
      await mocks.deletePolicy(name);
      set((state) => {
        const { [name]: _removed, ...policies } = state.policies;
        return { policies };
      });
    },
  }));
  return { useAppStore };
});

const { useAppStore } = await import("@/stores/app");
const seedPolicies = (policies: Record<string, Policy>) =>
  (
    useAppStore as unknown as {
      setState: (state: { policies: Record<string, Policy> }) => void;
    }
  ).setState({ policies });

const { MAX_REVIEW_AI_POLICY_BYTES, ReviewAIPolicyField, useReviewAIPolicyDraft } =
  await import("./ReviewAIPolicyField");

const reviewAIPolicy = (parent: string, content: string, enforce = true) =>
  create(PolicySchema, {
    name: `${parent}/policies/review_ai`,
    type: PolicyType.REVIEW_AI,
    enforce,
    policy: {
      case: "reviewAiPolicy",
      value: create(ReviewAIPolicySchema, { content }),
    },
  });

function Harness({ parentPath = PARENT }: { parentPath?: string }) {
  const policy = useReviewAIPolicyDraft({
    parentPath,
    resourceType: PolicyResourceType.PROJECT,
    can: (permission) => mocks.permissions[permission] ?? true,
  });
  const [saved, setSaved] = useState<boolean>();
  return (
    <>
      <ReviewAIPolicyField policy={policy} description="the description" />
      <output>{policy.isDirty ? "dirty" : "clean"}</output>
      <output>{`saved: ${saved}`}</output>
      <button type="button" onClick={() => void policy.save().then(setSaved)}>
        save
      </button>
      <button type="button" onClick={policy.revert}>
        revert
      </button>
    </>
  );
}

const editor = () => screen.findByRole("textbox", EDITOR);

const type = async (content: string) =>
  fireEvent.change(await editor(), { target: { value: content } });

const save = () =>
  act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "save" }));
  });

describe("ReviewAIPolicyField", () => {
  beforeEach(() => {
    mocks.permissions = {};
    mocks.unreadable.clear();
    mocks.fetchPolicy.mockReset();
    mocks.upsertPolicy.mockReset();
    mocks.deletePolicy.mockReset();
    seedPolicies({});
  });

  test("loads the stored policy into the editor", async () => {
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "No DROP on prod.") });
    render(<Harness />);

    const textbox = await editor();
    expect(textbox).toHaveValue("No DROP on prod.");
    expect(textbox).toHaveAccessibleDescription("the description");
    expect(mocks.fetchPolicy).toHaveBeenCalledWith({
      parentPath: PARENT,
      policyType: PolicyType.REVIEW_AI,
      refresh: false,
    });
    expect(screen.getByText("clean")).toBeInTheDocument();
  });

  test("a missing policy starts empty with example text, and a save creates it enforced", async () => {
    render(<Harness />);

    const textbox = await editor();
    expect(textbox).toHaveValue("");
    expect(textbox).toHaveAttribute(
      "placeholder",
      "sql-review.ai-review.policy.placeholder"
    );
    await type("Flag changes to the payments table.");
    expect(screen.getByText("dirty")).toBeInTheDocument();
    await save();

    expect(mocks.upsertPolicy).toHaveBeenCalledOnce();
    const { parentPath, policy } = mocks.upsertPolicy.mock.calls[0][0];
    expect(parentPath).toBe(PARENT);
    expect(policy.type).toBe(PolicyType.REVIEW_AI);
    expect(policy.resourceType).toBe(PolicyResourceType.PROJECT);
    expect(policy.enforce).toBe(true);
    expect(policy.policy.case).toBe("reviewAiPolicy");
    expect(policy.policy.value.content).toBe(
      "Flag changes to the payments table."
    );
    expect(screen.getByText("saved: true")).toBeInTheDocument();
    expect(screen.getByText("clean")).toBeInTheDocument();
    expect(await editor()).toHaveValue("Flag changes to the payments table.");
  });

  test("reverting drops the edit", async () => {
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "Stored.") });
    render(<Harness />);

    await type("Edited.");
    expect(screen.getByText("dirty")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button", { name: "revert" }));

    expect(await editor()).toHaveValue("Stored.");
    expect(screen.getByText("clean")).toBeInTheDocument();
  });

  test("typing the stored text back is not a change", async () => {
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "Stored.") });
    render(<Harness />);

    await type("Edited.");
    await type("Stored.");

    expect(screen.getByText("clean")).toBeInTheDocument();
  });

  test("a failed save keeps the edit", async () => {
    mocks.upsertPolicy.mockRejectedValueOnce(new Error("denied"));
    render(<Harness />);

    await type("Flag changes to the payments table.");
    await save();

    expect(await editor()).toHaveValue("Flag changes to the payments table.");
    expect(screen.getByText("dirty")).toBeInTheDocument();
    expect(screen.getByText("saved: false")).toBeInTheDocument();
  });

  test("text over the size limit, counted in UTF-8 bytes, shows an error and does not save", async () => {
    render(<Harness />);

    // Under the limit in characters, over it in bytes.
    const tooLarge = "表".repeat(Math.floor(MAX_REVIEW_AI_POLICY_BYTES / 3) + 1);
    expect(tooLarge.length).toBeLessThan(MAX_REVIEW_AI_POLICY_BYTES);
    await type(tooLarge);

    const textbox = await editor();
    expect(textbox).toHaveAttribute("aria-invalid", "true");
    // Exact byte counts, so a text one byte over never reads as the limit.
    const error = screen.getByRole("alert");
    expect(error).toHaveTextContent("sql-review.ai-review.policy.too-large");
    expect(error).toHaveTextContent(
      JSON.stringify({
        // Three UTF-8 bytes a character.
        size: (3 * tooLarge.length).toLocaleString(),
        limit: MAX_REVIEW_AI_POLICY_BYTES.toLocaleString(),
      })
    );
    expect(textbox).toHaveAccessibleDescription(
      /sql-review\.ai-review\.policy\.too-large/
    );
    await save();
    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(screen.getByText("saved: false")).toBeInTheDocument();

    await type("a".repeat(MAX_REVIEW_AI_POLICY_BYTES));
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(await editor()).toHaveAttribute("aria-invalid", "false");
  });

  test("clearing the text deletes the policy", async () => {
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "Stored.") });
    render(<Harness />);

    await type("  \n ");
    expect(screen.getByText("dirty")).toBeInTheDocument();
    await save();

    expect(mocks.upsertPolicy).not.toHaveBeenCalled();
    expect(mocks.deletePolicy).toHaveBeenCalledWith(POLICY_NAME);
    expect(screen.getByText("saved: true")).toBeInTheDocument();
    expect(screen.getByText("clean")).toBeInTheDocument();
    expect(await editor()).toHaveValue("");
  });

  test("clearing the text needs bb.policies.delete", async () => {
    mocks.permissions = { "bb.policies.delete": false };
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "Stored.") });
    render(<Harness />);

    await type("");

    expect(screen.getByRole("alert")).toHaveTextContent(
      "sql-review.ai-review.policy.cannot-clear"
    );
    await save();
    expect(mocks.deletePolicy).not.toHaveBeenCalled();
    expect(screen.getByText("saved: false")).toBeInTheDocument();
  });

  test("blank text without a policy is not a change", async () => {
    render(<Harness />);

    await type("   ");

    expect(screen.getByText("clean")).toBeInTheDocument();
  });

  // The server trims with Go's strings.TrimSpace, which differs from
  // JavaScript's trim on these two code points.
  test.each([
    ["U+0085, which the server trims, deletes", 0x85, "delete"],
    ["U+FEFF, which the server keeps, saves", 0xfeff, "upsert"],
  ])("text of only %s", async (_name, codePoint, expected) => {
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "Stored.") });
    render(<Harness />);

    await type(String.fromCodePoint(codePoint));
    await save();

    expect(mocks.deletePolicy).toHaveBeenCalledTimes(
      expected === "delete" ? 1 : 0
    );
    expect(mocks.upsertPolicy).toHaveBeenCalledTimes(
      expected === "upsert" ? 1 : 0
    );
  });

  test("shows that the policy is loading until it has been read", async () => {
    let finishRead = () => {};
    mocks.fetchPolicy.mockReturnValueOnce(
      new Promise<void>((resolve) => {
        finishRead = resolve;
      })
    );
    render(<Harness />);

    expect(screen.getByText("common.loading")).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
    await act(async () => finishRead());

    expect(await editor()).toBeInTheDocument();
    expect(screen.queryByText("common.loading")).not.toBeInTheDocument();
  });

  test("shows the policy as text without bb.policies.update", async () => {
    mocks.permissions = { "bb.policies.update": false };
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "No DROP on prod.") });
    render(<Harness />);

    expect(await screen.findByText("No DROP on prod.")).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  test("shows that there is no policy without bb.policies.create", async () => {
    mocks.permissions = { "bb.policies.create": false };
    render(<Harness />);

    expect(
      await screen.findByText("sql-review.ai-review.policy.none")
    ).toBeInTheDocument();
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();
  });

  test("editing an existing policy does not need bb.policies.create", async () => {
    mocks.permissions = { "bb.policies.create": false };
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "Stored.") });
    render(<Harness />);

    expect(await editor()).toBeEnabled();
  });

  test("says when the policy is paused, and an edit keeps it paused", async () => {
    seedPolicies({ [POLICY_NAME]: reviewAIPolicy(PARENT, "Stored.", false) });
    render(<Harness />);

    await type("Edited.");
    expect(screen.getByRole("alert")).toHaveTextContent(
      "sql-review.ai-review.policy.paused"
    );
    await save();

    expect(screen.getByRole("alert")).toHaveTextContent(
      "sql-review.ai-review.policy.paused"
    );
  });

  test("a failed read shows an error instead of the editor, and retry reads again", async () => {
    mocks.unreadable.add(POLICY_NAME);
    render(<Harness />);

    expect(await screen.findByRole("alert")).toHaveTextContent(
      "sql-review.ai-review.policy.load-failed"
    );
    expect(screen.queryByRole("textbox")).not.toBeInTheDocument();

    mocks.unreadable.clear();
    fireEvent.click(
      screen.getByRole("button", { name: "sql-review.standard-rules.retry" })
    );

    expect(await editor()).toBeInTheDocument();
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(mocks.fetchPolicy).toHaveBeenLastCalledWith({
      parentPath: PARENT,
      policyType: PolicyType.REVIEW_AI,
      refresh: true,
    });
  });

  test("moving to another project drops the edit", async () => {
    seedPolicies({
      [POLICY_NAME]: reviewAIPolicy(PARENT, "First."),
      "projects/q/policies/review_ai": reviewAIPolicy("projects/q", "Second."),
    });
    const { rerender } = render(<Harness />);
    await type("Edited.");

    rerender(<Harness parentPath="projects/q" />);

    expect(await editor()).toHaveValue("Second.");
    expect(screen.getByText("clean")).toBeInTheDocument();
  });
});
