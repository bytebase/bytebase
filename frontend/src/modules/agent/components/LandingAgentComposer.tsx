import { Send } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import type { LandingStarterId } from "../logic/landingStarters";
import {
  type AgentMentionRange,
  getMentionTriggerRange,
  hasAdditionalResourceMention,
  insertResourceMention,
  updateResourceMentionRange,
} from "../logic/resourceMention";
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
  const [stagedProject, setStagedProject] = useState<
    Pick<Project, "name" | "title"> | undefined
  >();
  const [selectedMention, setSelectedMention] =
    useState<AgentMentionRange | null>(null);
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
  }, [prompt, selectedMention]);

  const selectMentionProject = (project: Project) => {
    if (stagedProject && selectedMention) return;
    const label = project.title || project.name;
    const inserted = insertResourceMention(
      prompt,
      mentionRangeRef.current,
      label
    );
    if (!inserted) return;
    setPrompt(inserted.text);
    setSelectedMention(inserted.mention);
    setStagedProject({ name: project.name, title: label });
    pendingCaretRef.current = inserted.caret;
    mentionRangeRef.current = null;
    setResourcePickerOpen(false);
  };

  const hasAdditionalMention = Boolean(
    stagedProject &&
      selectedMention &&
      hasAdditionalResourceMention(prompt, selectedMention)
  );

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
    store.queueUserMessage({
      chatId: chat.id,
      text,
      page,
      ...(stagedProject
        ? {
            resourceReference: {
              kind: "project" as const,
              name: stagedProject.name,
              project: stagedProject.name,
              label: stagedProject.title,
            },
          }
        : {}),
    });
    store.open();
    setPrompt("");
    setStagedProject(undefined);
    setSelectedMention(null);
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
          mention={selectedMention}
          mentionLabel={stagedProject?.title}
          onDeleteMention={({ text, caret }) => {
            pendingCaretRef.current = caret;
            mentionRangeRef.current = null;
            setPrompt(text);
            setSelectedMention(null);
            setStagedProject(undefined);
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
            const nextMention = updateResourceMentionRange(
              prompt,
              nextText,
              selectedMention,
              stagedProject?.title ?? ""
            );
            setPrompt(nextText);
            setSelectedMention(nextMention);
            if (!nextMention) setStagedProject(undefined);
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
                setResourcePickerOpen(false);
                mentionRangeRef.current = null;
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
          onOpenChange={setResourcePickerOpen}
          anchorRef={textareaRef}
          workspaceName={workspaceName}
          hasSelectedReference={Boolean(stagedProject && selectedMention)}
          onSelectProject={selectMentionProject}
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
