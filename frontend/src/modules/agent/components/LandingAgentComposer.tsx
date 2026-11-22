import { Send } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { projectNamePrefix } from "@/lib/resourceName";
import type { LandingStarterId } from "../logic/landingStarters";
import { isAgentSendShortcut } from "../logic/sendShortcut";
import type { AgentDraft } from "../logic/types";
import { useAgentStore } from "../store/agent";
import { AgentComposer } from "./AgentComposer";
import { AgentMentionTextarea } from "./AgentMentionTextarea";
import { AgentResourcePicker } from "./AgentResourcePicker";
import { LandingStarterSelector } from "./LandingStarterSelector";
import { useAgentResourceDraft } from "./useAgentResourceDraft";

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
  const sendWithEnter = useAgentStore((s) => s.preferences.sendWithEnter);
  const [draft, setDraft] = useState<AgentDraft>({ text: "" });
  const {
    textareaRef,
    mentions,
    references,
    hasInvalidReferences,
    hasUnboundMention,
    onChange,
    onDeleteMention,
    handleKeyDown,
    pickerProps,
  } = useAgentResourceDraft({ draft, onDraftChange: setDraft });
  const prompt = draft.text;

  const selectChat = () => {
    const store = useAgentStore.getState();
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
    if (!text || hasUnboundMention || hasInvalidReferences) return;
    const selected = selectChat();
    if (!selected) return;
    const { store, chat, page } = selected;
    store.queueUserMessage({
      chatId: chat.id,
      text,
      page,
      ...(references[0] ? { resourceReference: references[0] } : {}),
      ...(references.length > 1 ? { resourceReferences: references } : {}),
    });
    store.open();
    setDraft({ text: "" });
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
      className="min-w-0"
      onSubmit={(event) => {
        event.preventDefault();
        sendMessage();
      }}
    >
      <AgentComposer
        header={
          <LandingStarterSelector
            starterIds={starterIds}
            onSelect={sendStarter}
          />
        }
        footer={
          <span className="min-w-0 max-w-full truncate">
            {t("agent.context")}: {contextTitle}
          </span>
        }
        action={
          <Button
            type="submit"
            size="sm"
            disabled={
              !prompt.trim() || hasUnboundMention || hasInvalidReferences
            }
          >
            <Send className="size-4" />
            {t("agent.send")}
          </Button>
        }
      >
        <AgentMentionTextarea
          textareaRef={textareaRef}
          mentions={mentions.map(({ resource, range }) => ({
            range,
            label: resource.label,
          }))}
          onDeleteMention={onDeleteMention}
          size="md"
          rows={2}
          value={prompt}
          aria-label={placeholder}
          placeholder={`${placeholder} · ${sendWithEnter ? t("agent.input-keyboard-hint") : t("agent.input-keyboard-hint-newline")}`}
          className="min-h-20 resize-none border-0 focus:border-0 focus:ring-0"
          onChange={onChange}
          onKeyDown={(event) => {
            if (handleKeyDown(event)) return;
            if (isAgentSendShortcut(event.nativeEvent, sendWithEnter)) {
              event.preventDefault();
              sendMessage();
            }
          }}
        />
        <AgentResourcePicker
          {...pickerProps}
          workspaceName={workspaceName}
          projectName={
            contextName.startsWith(projectNamePrefix) ? contextName : undefined
          }
        />
      </AgentComposer>
    </form>
  );
}
