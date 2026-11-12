import { create } from "@bufbuild/protobuf";
import { act, renderHook } from "@testing-library/react";
import { type ChangeEvent, type KeyboardEvent, useState } from "react";
import { describe, expect, test, vi } from "vitest";
import { type Issue, IssueSchema } from "@/types/proto-es/v1/issue_service_pb";
import { type Plan, PlanSchema } from "@/types/proto-es/v1/plan_service_pb";
import { ProjectSchema } from "@/types/proto-es/v1/project_service_pb";
import type { AgentDraft, AgentResourceReference } from "../logic/types";
import { useAgentResourceDraft } from "./useAgentResourceDraft";

const project: AgentResourceReference = {
  kind: "project",
  name: "projects/orders",
  project: "projects/orders",
  label: "Orders",
};
const database: AgentResourceReference = {
  kind: "database",
  name: "instances/main/databases/orders",
  project: project.name,
  label: "orders (Main)",
};

function renderDraft(
  initial: AgentDraft = { text: "" },
  initialOptions: { enabled?: boolean; boundReferences?: AgentResourceReference[] } = {}
) {
  const onDraftChange = vi.fn();
  const hook = renderHook(
    (options: typeof initialOptions) => {
      const [draft, setDraft] = useState(initial);
      return {
        draft,
        ...useAgentResourceDraft({
          draft,
          onDraftChange: (next) => {
            onDraftChange(next);
            setDraft(next);
          },
          ...options,
        }),
      };
    },
    { initialProps: initialOptions }
  );
  const change = (text: string, caret = text.length) =>
    act(() =>
      hook.result.current.onChange({
        target: { value: text, selectionStart: caret },
      } as ChangeEvent<HTMLTextAreaElement>)
    );
  return { ...hook, change, onDraftChange };
}

describe("useAgentResourceDraft", () => {
  test("tracks @ selection as a controlled resource mention", () => {
    const { result, change, onDraftChange } = renderDraft();
    change("Check @");
    expect(result.current.pickerProps.open).toBe(true);

    act(() =>
      result.current.pickerProps.onSelectProject(
        create(ProjectSchema, { name: project.name, title: project.label })
      )
    );

    expect(result.current.draft).toEqual({
      text: "Check @Orders ",
      mentions: [{ resource: project, range: { start: 6, end: 13 } }],
    });
    expect(result.current.mentions).toEqual(result.current.draft.mentions);
    expect(result.current.references).toEqual([project]);
    expect(result.current.pickerProps.open).toBe(false);
    expect(onDraftChange).toHaveBeenCalledTimes(2);

    change("Please check @Orders ");
    expect(result.current.mentions[0].range).toEqual({ start: 13, end: 20 });
  });

  test("removes only the current trigger when canceled", () => {
    const { result, change } = renderDraft({
      text: "@Orders ",
      mentions: [{ resource: project, range: { start: 0, end: 7 } }],
    });
    change("@Orders check @later text", 20);
    expect(result.current.pickerProps.open).toBe(true);
    expect(result.current.hasUnboundMention).toBe(true);

    act(() => result.current.pickerProps.onOpenChange(false));

    expect(result.current.draft.text).toBe("@Orders check  text");
    expect(result.current.mentions).toHaveLength(1);
    expect(result.current.hasUnboundMention).toBe(false);
  });

  test("removes selected mentions without leaving stale metadata", () => {
    const { result } = renderDraft({
      text: "@Orders check",
      mentions: [{ resource: project, range: { start: 0, end: 7 } }],
    });
    act(() => result.current.onDeleteMention({ text: "check", caret: 0 }));
    expect(result.current.draft).toEqual({ text: "check", mentions: [] });
    expect(result.current.references).toEqual([]);
    expect(result.current.pickerProps.selectedProjectName).toBeUndefined();
  });

  test("does not treat a bare @ in otherwise unbound text as an additional mention", () => {
    const { result, change } = renderDraft();
    change("Ask @someone");
    expect(result.current.hasUnboundMention).toBe(false);
  });

  test("preserves legacy draft references when edited", () => {
    const { result, change } = renderDraft({
      text: "@Orders ",
      resource: project,
      mention: { start: 0, end: 7 },
    });
    change("@Orders describe");
    expect(result.current.draft).toEqual({
      text: "@Orders describe",
      mentions: [{ resource: project, range: { start: 0, end: 7 } }],
    });
  });

  test.each([
    {
      kind: "issue" as const,
      resource: create(IssueSchema, {
        name: "projects/orders/issues/42",
        title: "Review index",
      }),
      label: "Review index",
    },
    {
      kind: "plan" as const,
      resource: create(PlanSchema, {
        name: "projects/orders/plans/42",
        title: "Review index",
      }),
      label: "Review index (#42)",
    },
  ])("converts selected $kind metadata with the shared label", ({ kind, resource, label }) => {
    const { result, change } = renderDraft();
    change("@");
    act(() => {
      if (kind === "issue") {
        result.current.pickerProps.onSelectIssue(resource as Issue);
      } else {
        result.current.pickerProps.onSelectPlan(resource as Plan);
      }
    });
    expect(result.current.references).toEqual([
      { kind, name: resource.name, project: project.name, label },
    ]);
  });

  test("bound resources constrain the picker and cannot be replaced by another resource", () => {
    const { result, change, onDraftChange } = renderDraft(
      { text: "" },
      { boundReferences: [database] }
    );
    expect(result.current.references).toEqual([database]);
    expect(result.current.pickerProps.selectedProjectName).toBe(project.name);
    expect(result.current.pickerProps.hasSelectedReference).toBe(true);
    change("@");
    expect(result.current.hasUnboundMention).toBe(true);
    const edits = onDraftChange.mock.calls.length;
    act(() =>
      result.current.pickerProps.onSelectPlan(
        create(PlanSchema, { name: "projects/orders/plans/7", title: "Other" })
      )
    );
    expect(onDraftChange).toHaveBeenCalledTimes(edits);
    expect(result.current.references).toEqual([database]);
  });

  test("blocks a conflicting resource staged before the bound resource resolves", () => {
    const planReference: AgentResourceReference = {
      kind: "plan",
      name: "projects/orders/plans/7",
      project: project.name,
      label: "Index",
    };
    const { result, rerender } = renderDraft({
      text: "@Index describe",
      mentions: [{ resource: planReference, range: { start: 0, end: 6 } }],
    });
    expect(result.current.hasInvalidReferences).toBe(false);

    rerender({ boundReferences: [database] });

    expect(result.current.references).toEqual([]);
    expect(result.current.hasInvalidReferences).toBe(true);
    expect(result.current.mentions).toHaveLength(1);
    expect(result.current.draft.text).toBe("@Index describe");
  });

  test("blocks a stale draft child from a different bound project", () => {
    const foreignReference: AgentResourceReference = {
      kind: "plan",
      name: "projects/other/plans/7",
      project: "projects/other",
      label: "Index",
    };
    const { result, change } = renderDraft(
      {
        text: "@Index describe",
        mentions: [{ resource: foreignReference, range: { start: 0, end: 6 } }],
      },
      { boundReferences: [project] }
    );
    expect(result.current.references).toEqual([]);
    expect(result.current.hasInvalidReferences).toBe(true);

    change("describe");

    expect(result.current.references).toEqual([project]);
    expect(result.current.hasInvalidReferences).toBe(false);
  });

  test("does not silently discard stale incompatible draft mentions", () => {
    const foreignReference: AgentResourceReference = {
      kind: "plan",
      name: "projects/other/plans/7",
      project: "projects/other",
      label: "Index",
    };
    const { result } = renderDraft({
      text: "@Orders @Index describe",
      mentions: [
        { resource: project, range: { start: 0, end: 7 } },
        { resource: foreignReference, range: { start: 8, end: 14 } },
      ],
    });

    expect(result.current.references).toEqual([]);
    expect(result.current.hasInvalidReferences).toBe(true);
  });

  test("deduplicates a selected reference already bound to the context", () => {
    const { result, change } = renderDraft(
      { text: "" },
      { boundReferences: [project] }
    );
    change("@");
    act(() =>
      result.current.pickerProps.onSelectProject(
        create(ProjectSchema, { name: project.name, title: project.label })
      )
    );
    expect(result.current.mentions).toHaveLength(1);
    expect(result.current.references).toEqual([project]);
    change("@Orders @");
    act(() =>
      result.current.pickerProps.onSelectProject(
        create(ProjectSchema, { name: project.name, title: project.label })
      )
    );
    expect(result.current.mentions).toHaveLength(1);
    expect(result.current.draft.text).toBe("@Orders @");
  });

  test("allows a same-project child but rejects a foreign project with a bound project", () => {
    const { result, change } = renderDraft(
      { text: "" },
      { boundReferences: [project] }
    );
    change("@");
    act(() =>
      result.current.pickerProps.onSelectPlan(
        create(PlanSchema, { name: "projects/other/plans/7", title: "Other" })
      )
    );
    expect(result.current.mentions).toHaveLength(0);
    act(() =>
      result.current.pickerProps.onSelectPlan(
        create(PlanSchema, { name: "projects/orders/plans/7", title: "Index" })
      )
    );
    expect(result.current.references).toEqual([
      project,
      { kind: "plan", name: "projects/orders/plans/7", project: project.name, label: "Index (#7)" },
    ]);
  });

  test("disabled resource picking does not open or insert resources", () => {
    const { result, change } = renderDraft({ text: "" }, { enabled: false });
    change("@");
    expect(result.current.draft.text).toBe("@");
    expect(result.current.pickerProps.open).toBe(false);
    act(() =>
      result.current.pickerProps.onSelectProject(
        create(ProjectSchema, { name: project.name, title: project.label })
      )
    );
    expect(result.current.mentions).toHaveLength(0);
  });

  test.each([
    { key: "Enter" },
    { key: "Enter", ctrlKey: true },
    { key: "Enter", metaKey: true },
    { key: "Escape" },
    { key: "ArrowDown" },
  ])("suppresses caller send handling while selecting with %j", (modifiers) => {
    const { result, change } = renderDraft();
    change("@");
    const event = {
      ...modifiers,
      nativeEvent: {},
      currentTarget: { selectionStart: 1 },
      preventDefault: vi.fn(),
    } as unknown as KeyboardEvent<HTMLTextAreaElement>;
    let handled = false;
    act(() => { handled = result.current.handleKeyDown(event); });
    expect(handled).toBe(true);
    if (modifiers.key === "Escape") {
      expect(result.current.draft.text).toBe("");
      expect(result.current.pickerProps.open).toBe(false);
    }
  });

  test.each([{ isComposing: true }, { keyCode: 229 }])("suppresses caller send handling for IME %j", (nativeEvent) => {
    const { result } = renderDraft();
    expect(result.current.handleKeyDown({ nativeEvent } as unknown as KeyboardEvent<HTMLTextAreaElement>)).toBe(true);
  });

  test("allows normal send handling once the caret leaves the trigger", () => {
    const { result, change } = renderDraft();
    change("@");
    let handled = true;
    act(() => {
      handled = result.current.handleKeyDown({
        key: "Enter",
        nativeEvent: {},
        currentTarget: { selectionStart: 0 },
      } as KeyboardEvent<HTMLTextAreaElement>);
    });
    expect(handled).toBe(false);
    expect(result.current.pickerProps.open).toBe(false);
  });
});
