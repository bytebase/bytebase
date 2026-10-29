import { Send } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { projectNamePrefix } from "@/lib/resourceName";
import {
  getNameParentTokens,
  issueNamePrefix,
} from "@/stores/modules/v1/common";
import type { Database } from "@/types/proto-es/v1/database_service_pb";
import type { Issue } from "@/types/proto-es/v1/issue_service_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { LandingStarterId } from "../logic/landingStarters";
import {
  type AgentMentionRange,
  canAddResourceMention,
  getDatabaseMentionLabel,
  getMentionTriggerRange,
  hasUnboundResourceMention,
  insertResourceMention,
  normalizeAgentResourceReferences,
  removeMentionTrigger,
  updateDraftResourceMentions,
} from "../logic/resourceMention";
import type {
  AgentResourceMention,
  AgentResourceReference,
} from "../logic/types";
import { selectHasRunningChat, useAgentStore } from "../store/agent";
import { AgentMentionTextarea } from "./AgentMentionTextarea";
import { AgentResourcePicker } from "./AgentResourcePicker";
import { LandingStarterSelector } from "./LandingStarterSelector";

interface LandingAgentComposerProps {
  contextName: string;
  contextTitle: string;
  workspaceName: string;
  placeholder: string;
  starterIds: LandingStarterId[];
}

export function LandingAgentComposer({
  contextName,
  contextTitle,
  workspaceName,
  placeholder,
  starterIds,
}: LandingAgentComposerProps) {
  const { t } = useTranslation();
  const [prompt, setPrompt] = useState("");
  const [stagedMentions, setStagedMentions] = useState<AgentResourceMention[]>(
    []
  );
  const [resourcePickerOpen, setResourcePickerOpen] = useState(false);
  const mentionRangeRef = useRef<AgentMentionRange | null>(null);
  const pendingCaretRef = useRef<number | null>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const hasRunningChat = useAgentStore(selectHasRunningChat);
  const hasQueuedMessage = useAgentStore(
    (state) => state.queuedUserMessage !== null
  );

  useEffect(() => {
    if (pendingCaretRef.current === null) return;
    textareaRef.current?.focus();
    textareaRef.current?.setSelectionRange(
      pendingCaretRef.current,
      pendingCaretRef.current
    );
    pendingCaretRef.current = null;
  }, [prompt, stagedMentions]);

  const selectMentionResource = (resource: AgentResourceReference) => {
    if (!canAddResourceMention(stagedMentions, resource)) return;
    const label = resource.label;
    const inserted = insertResourceMention(
      prompt,
      mentionRangeRef.current,
      label
    );
    if (!inserted) return;
    setPrompt(inserted.text);
    setStagedMentions([
      ...updateDraftResourceMentions(prompt, inserted.text, stagedMentions),
      { resource, range: inserted.mention },
    ]);
    pendingCaretRef.current = inserted.caret;
    mentionRangeRef.current = null;
    setResourcePickerOpen(false);
  };

  const selectMentionProject = (project: Project) =>
    selectMentionResource({
      kind: "project",
      name: project.name,
      project: project.name,
      label: project.title || project.name,
    });

  const selectMentionDatabase = (database: Database) =>
    selectMentionResource({
      kind: "database",
      name: database.name,
      project: database.project,
      label: getDatabaseMentionLabel(database),
    });

  const selectMentionIssue = (issue: Issue) => {
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
  };

  const handleResourcePickerOpenChange = (nextOpen: boolean) => {
    setResourcePickerOpen(nextOpen);
    if (nextOpen) return;
    const trigger = mentionRangeRef.current;
    mentionRangeRef.current = null;
    const removed = removeMentionTrigger(prompt, trigger);
    if (!removed) return;
    pendingCaretRef.current = removed.caret;
    setStagedMentions(
      updateDraftResourceMentions(prompt, removed.text, stagedMentions)
    );
    setPrompt(removed.text);
  };

  const selectedProjectMention = stagedMentions.find(
    ({ resource }) => resource.kind === "project"
  );
  const hasAdditionalMention =
    stagedMentions.length > 0 &&
    hasUnboundResourceMention(prompt, stagedMentions);

  const selectChat = () => {
    const store = useAgentStore.getState();
    if (selectHasRunningChat(store) || store.queuedUserMessage) return;
    const page = {
      path: `${window.location.pathname}${window.location.search}${window.location.hash}`,
      title: document.title,
    };
    const chat = store.selectOrCreateChatForContext(
      contextName,
      page,
      workspaceName
    );
    if (!chat) return;
    return { store, chat, page };
  };

  const sendMessage = () => {
    const text = prompt.trim();
    if (!text || hasAdditionalMention) return;
    const selected = selectChat();
    if (!selected) return;
    const { store, chat, page } = selected;
    const references = normalizeAgentResourceReferences(
      stagedMentions.map(({ resource }) => resource)
    );
    store.queueUserMessage({
      chatId: chat.id,
      text,
      page,
      ...(references[0] ? { resourceReference: references[0] } : {}),
      ...(references.length > 1 ? { resourceReferences: references } : {}),
    });
    store.open();
    setPrompt("");
    setStagedMentions([]);
  };

  const sendStarter = (starterId: LandingStarterId) => {
    const selected = selectChat();
    if (!selected) return;
    const { store, chat, page } = selected;
    store.queueUserMessage({
      chatId: chat.id,
      text: t(`agent.landing-starters.${starterId}`),
      page,
      runMode: "read-only",
      starterId,
    });
    store.open();
  };

  return (
    <form
      className="min-w-0 rounded-sm border border-accent/60 bg-background focus-within:border-accent focus-within:ring-1 focus-within:ring-accent"
      onSubmit={(event) => {
        event.preventDefault();
        sendMessage();
      }}
    >
      <div className="px-3 pt-2">
        <LandingStarterSelector
          starterIds={starterIds}
          disabled={hasRunningChat || hasQueuedMessage}
          onSelect={sendStarter}
        />
      </div>
      <div className="relative">
        <AgentMentionTextarea
          textareaRef={textareaRef}
          mentions={stagedMentions.map(({ resource, range }) => ({
            range,
            label: resource.label,
          }))}
          onDeleteMention={({ text, caret }) => {
            pendingCaretRef.current = caret;
            mentionRangeRef.current = null;
            setStagedMentions(
              updateDraftResourceMentions(prompt, text, stagedMentions)
            );
            setPrompt(text);
            setResourcePickerOpen(false);
          }}
          size="md"
          rows={2}
          value={prompt}
          aria-label={placeholder}
          placeholder={`${placeholder} · ${t("agent.input-keyboard-hint")}`}
          className="min-h-20 resize-none border-0 focus:border-0 focus:ring-0"
          onChange={(event) => {
            const nextText = event.target.value;
            const nextMentions = updateDraftResourceMentions(
              prompt,
              nextText,
              stagedMentions
            );
            setPrompt(nextText);
            setStagedMentions(nextMentions);
            const range = getMentionTriggerRange(
              nextText,
              event.target.selectionStart
            );
            mentionRangeRef.current = range;
            setResourcePickerOpen(Boolean(range));
          }}
          onKeyDown={(event) => {
            if (event.nativeEvent.isComposing) return;
            if (resourcePickerOpen) {
              const range = getMentionTriggerRange(
                prompt,
                event.currentTarget.selectionStart
              );
              if (!range) {
                setResourcePickerOpen(false);
                mentionRangeRef.current = null;
              } else if (event.key === "Escape") {
                event.preventDefault();
                handleResourcePickerOpenChange(false);
                return;
              } else if (
                event.key === "Enter" &&
                !event.shiftKey &&
                !event.altKey &&
                !event.ctrlKey &&
                !event.metaKey
              ) {
                event.preventDefault();
                return;
              }
            }
            if (
              event.key === "Enter" &&
              !event.shiftKey &&
              !event.altKey &&
              !event.ctrlKey &&
              !event.metaKey
            ) {
              event.preventDefault();
              sendMessage();
            }
          }}
        />
        <AgentResourcePicker
          open={resourcePickerOpen}
          onOpenChange={handleResourcePickerOpenChange}
          anchorRef={textareaRef}
          workspaceName={workspaceName}
          projectName={
            contextName.startsWith(projectNamePrefix) ? contextName : undefined
          }
          selectedProjectName={selectedProjectMention?.resource.name}
          hasSelectedReference={
            stagedMentions.length > 1 ||
            (stagedMentions.length === 1 && !selectedProjectMention)
          }
          onSelectProject={selectMentionProject}
          onSelectDatabase={selectMentionDatabase}
          onSelectIssue={selectMentionIssue}
        />
      </div>
      <div className="flex flex-wrap items-end justify-between gap-2 px-3 pb-3">
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2 text-xs text-control-light">
          <span className="min-w-0 max-w-full truncate">
            {t("agent.context")}: {contextTitle}
          </span>
        </div>
        <Button
          type="submit"
          size="sm"
          disabled={
            !prompt.trim() ||
            hasRunningChat ||
            hasQueuedMessage ||
            hasAdditionalMention
          }
        >
          <Send className="size-4" />
          {t("agent.send")}
        </Button>
      </div>
    </form>
  );
}
