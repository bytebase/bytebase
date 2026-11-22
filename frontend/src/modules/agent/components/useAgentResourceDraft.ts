import {
  type ChangeEventHandler,
  type ComponentProps,
  type KeyboardEvent,
  useEffect,
  useRef,
  useState,
} from "react";
import { projectNamePrefix } from "@/lib/resourceName";
import {
  getNameParentTokens,
  issueNamePrefix,
  planNamePrefix,
} from "@/stores/modules/v1/common";
import {
  getDatabaseMentionLabel,
  getDraftResourceMentions,
  getMentionTriggerRange,
  getPlanMentionLabel,
  hasUnboundResourceMention,
  insertResourceMention,
  normalizeAgentResourceReferences,
  removeMentionTrigger,
  updateDraftResourceMentions,
} from "../logic/resourceMention";
import type {
  AgentDraft,
  AgentMentionRange,
  AgentResourceReference,
} from "../logic/types";
import type { AgentResourcePicker } from "./AgentResourcePicker";

interface AgentResourceDraftOptions {
  draft: AgentDraft;
  onDraftChange: (draft: AgentDraft) => void;
  enabled?: boolean;
  boundReferences?: AgentResourceReference[];
}

type ResourcePickerProps = Omit<
  ComponentProps<typeof AgentResourcePicker>,
  "workspaceName" | "projectName"
>;

const combineReferences = (
  boundReferences: AgentResourceReference[],
  draftReferences: AgentResourceReference[]
): AgentResourceReference[] => {
  const references = [...boundReferences];
  for (const resource of draftReferences) {
    if (
      references.some(
        (existing) =>
          existing.kind === resource.kind &&
          existing.name === resource.name &&
          existing.project === resource.project
      )
    )
      continue;
    references.push(resource);
  }
  return normalizeAgentResourceReferences(references);
};

export function useAgentResourceDraft({
  draft,
  onDraftChange,
  enabled = true,
  boundReferences = [],
}: AgentResourceDraftOptions) {
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const mentionRangeRef = useRef<AgentMentionRange | null>(null);
  const pendingCaretRef = useRef<number | null>(null);
  const [resourcePickerOpen, setResourcePickerOpen] = useState(false);
  const mentions = getDraftResourceMentions(draft);
  const references = combineReferences(
    boundReferences,
    mentions.map(({ resource }) => resource)
  );
  const hasInvalidReferences =
    ((boundReferences.length > 0 || mentions.length > 0) &&
      references.length === 0) ||
    (draft.mentions !== undefined && draft.mentions.length !== mentions.length);

  useEffect(() => {
    if (pendingCaretRef.current === null) return;
    textareaRef.current?.focus();
    textareaRef.current?.setSelectionRange(
      pendingCaretRef.current,
      pendingCaretRef.current
    );
    pendingCaretRef.current = null;
  }, [draft]);

  const selectMentionResource = (resource: AgentResourceReference) => {
    if (
      !enabled ||
      mentions.some(
        (mention) =>
          mention.resource.kind === resource.kind &&
          mention.resource.name === resource.name
      )
    )
      return;
    const nextReferences = combineReferences(boundReferences, [
      ...mentions.map((mention) => mention.resource),
      resource,
    ]);
    if (nextReferences.length === 0) return;
    const inserted = insertResourceMention(
      draft.text,
      mentionRangeRef.current,
      resource.label
    );
    if (!inserted) return;
    pendingCaretRef.current = inserted.caret;
    mentionRangeRef.current = null;
    setResourcePickerOpen(false);
    onDraftChange({
      text: inserted.text,
      mentions: [
        ...updateDraftResourceMentions(draft.text, inserted.text, mentions),
        { resource, range: inserted.mention },
      ],
    });
  };

  const handleResourcePickerOpenChange = (nextOpen: boolean) => {
    setResourcePickerOpen(enabled && nextOpen);
    if (nextOpen) return;
    const trigger = mentionRangeRef.current;
    mentionRangeRef.current = null;
    const removed = removeMentionTrigger(draft.text, trigger);
    if (!removed) return;
    pendingCaretRef.current = removed.caret;
    onDraftChange({
      text: removed.text,
      mentions: updateDraftResourceMentions(draft.text, removed.text, mentions),
    });
  };

  const onChange: ChangeEventHandler<HTMLTextAreaElement> = (event) => {
    const nextText = event.target.value;
    onDraftChange({
      text: nextText,
      mentions: updateDraftResourceMentions(draft.text, nextText, mentions),
    });
    const range = enabled
      ? getMentionTriggerRange(nextText, event.target.selectionStart)
      : null;
    mentionRangeRef.current = range;
    setResourcePickerOpen(Boolean(range));
  };

  const onDeleteMention = ({
    text,
    caret,
  }: {
    text: string;
    caret: number;
  }) => {
    pendingCaretRef.current = caret;
    mentionRangeRef.current = null;
    setResourcePickerOpen(false);
    onDraftChange({
      text,
      mentions: updateDraftResourceMentions(draft.text, text, mentions),
    });
  };

  const handleKeyDown = (
    event: KeyboardEvent<HTMLTextAreaElement>
  ): boolean => {
    if (event.nativeEvent.isComposing || event.nativeEvent.keyCode === 229)
      return true;
    if (!resourcePickerOpen || !enabled) return false;
    const range = getMentionTriggerRange(
      draft.text,
      event.currentTarget.selectionStart
    );
    if (!range) {
      setResourcePickerOpen(false);
      mentionRangeRef.current = null;
      return false;
    }
    if (event.key === "Escape") {
      event.preventDefault();
      handleResourcePickerOpenChange(false);
    } else if (
      event.key === "Enter" &&
      !event.shiftKey &&
      !event.altKey &&
      !event.ctrlKey &&
      !event.metaKey
    ) {
      event.preventDefault();
    }
    return true;
  };

  const pickerProps: ResourcePickerProps = {
    open: enabled && resourcePickerOpen,
    onOpenChange: handleResourcePickerOpenChange,
    anchorRef: textareaRef,
    selectedProjectName:
      references.find((resource) => resource.kind === "project")?.name ??
      boundReferences[0]?.project,
    hasSelectedReference: references.some(
      (resource) => resource.kind !== "project"
    ),
    onSelectProject: (project) =>
      selectMentionResource({
        kind: "project",
        name: project.name,
        project: project.name,
        label: project.title || project.name,
      }),
    onSelectDatabase: (database) =>
      selectMentionResource({
        kind: "database",
        name: database.name,
        project: database.project,
        label: getDatabaseMentionLabel(database),
      }),
    onSelectIssue: (issue) => {
      const [projectId] = getNameParentTokens(issue.name, [
        projectNamePrefix,
        issueNamePrefix,
      ]);
      if (!projectId) return;
      selectMentionResource({
        kind: "issue",
        name: issue.name,
        project: `${projectNamePrefix}${projectId}`,
        label: issue.title || issue.name,
      });
    },
    onSelectPlan: (plan) => {
      const [projectId, planId] = getNameParentTokens(plan.name, [
        projectNamePrefix,
        planNamePrefix,
      ]);
      if (!projectId || !planId) return;
      selectMentionResource({
        kind: "plan",
        name: plan.name,
        project: `${projectNamePrefix}${projectId}`,
        label: getPlanMentionLabel(plan),
      });
    },
  };

  return {
    textareaRef,
    mentions,
    references,
    hasInvalidReferences,
    hasUnboundMention:
      references.length > 0 && hasUnboundResourceMention(draft.text, mentions),
    onChange,
    onDeleteMention,
    handleKeyDown,
    pickerProps,
  };
}
