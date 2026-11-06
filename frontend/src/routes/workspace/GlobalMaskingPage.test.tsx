import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const {
  state,
  guard,
  fixtures,
  parsedExpr,
  parseCondition,
  validateCondition,
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
  guard: vi.fn(),
  state: {
    environmentList: [],
    getOrFetchPolicyByParentAndType: vi.fn(),
    getOrFetchSettingByName: vi.fn(async () => undefined),
    hasInstanceFeature: vi.fn(() => true),
    hasPermission: true,
    settingsByName: {},
    upsertPolicy: vi.fn(async (_request: unknown) => undefined),
    workspaceResourceName: () => "workspaces/default",
  },
}));

vi.mock("@/modules/cel", () => ({
  ExprType: { RawString: "RawString" },
  buildCELExpr: vi.fn(async () => ({})),
  emptySimpleExpr: () => parsedExpr,
  resolveCELExpr: () => parsedExpr,
  validateSimpleExpr: validateCondition,
  wrapAsGroup: (expr: unknown) => expr,
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
      >
        {JSON.stringify(expr)}
      </output>
      {!readonly && (
        <button
          type="button"
          onClick={() =>
            onUpdate({
              ...parsedExpr,
              args: [{ type: "RawString", content: "true" }],
            })
          }
        >
          Change conditions
        </button>
      )}
    </>
  ),
}));
vi.mock("@/components/FeatureAttention", () => ({
  FeatureAttention: () => null,
}));
vi.mock("@/modules/agent/components/AgentPageHelpButton", () => ({
  AgentPageHelpButton: () => (
    <button type="button">Ask Agent about masking</button>
  ),
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
    ],
  }),
}));
vi.mock("@/hooks/useUnsavedChangesGuard", () => ({
  useUnsavedChangesGuard: guard,
}));
vi.mock("@/lib/sensitive-data/components-utils", () => ({
  factorOperatorOverrideMap: {},
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
  Setting_SettingName: { DATA_CLASSIFICATION: 2, SEMANTIC_TYPES: 1 },
}));
vi.mock("@/types/proto-es/v1/subscription_service_pb", () => ({
  PlanFeature: { FEATURE_DATA_MASKING: 1 },
}));
vi.mock("@bufbuild/protobuf", () => ({
  create: (_schema: unknown, init?: Record<string, unknown>) => ({ ...init }),
}));
vi.mock("uuid", () => ({ v4: () => "new-rule" }));
vi.mock("@/utils", () => ({
  arraySwap: (items: unknown[], a: number, b: number) => {
    [items[a], items[b]] = [items[b], items[a]];
  },
  batchConvertCELStringToParsedExpr: parseCondition,
  batchConvertParsedExprToCELString: vi.fn(async () => [
    'resource.environment_id in ["test"]',
  ]),
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
const prefix = "settings.sensitive-data.global-rules.";

beforeEach(() => {
  vi.clearAllMocks();
  parseCondition
    .mockReset()
    .mockResolvedValue([{ exprKind: { case: "callExpr" } }]);
  state.hasPermission = true;
  validateCondition.mockReturnValue(true);
  state.hasInstanceFeature.mockReturnValue(true);
  state.upsertPolicy.mockReset().mockResolvedValue(undefined);
  state.getOrFetchPolicyByParentAndType.mockResolvedValue({
    policy: {
      case: "maskingRulePolicy",
      value: { rules: structuredClone(fixtures) },
    },
  });
});
afterEach(cleanup);
async function renderPage() {
  render(<GlobalMaskingPage />);
  await screen.findByText("Test masking");
}
async function editFirstRule() {
  await renderPage();
  fireEvent.click(
    screen.getAllByRole("button", { name: prefix + "edit-rule" })[0]
  );
  return await screen.findByRole("textbox", { name: prefix + "name" });
}
function savedRules() {
  const request = state.upsertPolicy.mock.calls.at(-1)?.[0] as {
    policy: { policy: { value: { rules: typeof fixtures } } };
  };
  return request.policy.policy.value.rules;
}

describe("GlobalMaskingPage", () => {
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
