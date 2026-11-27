import {
  ArrowUpRight,
  BadgeCheck,
  CircleDot,
  ClipboardList,
  Database,
  Folder,
  LoaderCircle,
  Server,
  Shield,
  Tags,
  Users,
} from "lucide-react";
import { useRef } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { projectNamePrefix } from "@/lib/resourceName";
import type { AgentPageResource } from "../logic/chatContext";
import { isAgentSendShortcut } from "../logic/sendShortcut";
import type { AgentSqlContextCapture } from "../logic/sqlEditorContext";
import type {
  AgentDraft,
  AgentResourceReference,
  AgentRunMode,
} from "../logic/types";
import { useAgentStore } from "../store/agent";
import { AgentAIConfigurationRecovery } from "./AgentAIConfigurationRecovery";
import { AgentChatbox, AgentChatboxFooter } from "./AgentChatbox";
import { AgentResourcePicker } from "./AgentResourcePicker";
import { useAgentResourceDraft } from "./useAgentResourceDraft";
import {
  type AgentSqlSuggestionSource,
  useSqlEditorSuggestions,
} from "./useSqlEditorSuggestions";

type AgentSqlTemplateStarterId =
  | "write-query"
  | "find-database"
  | "fix-error"
  | "explore-database"
  | "table-relationships"
  | "explain-selection"
  | "review-selection"
  | "improve-selection";

export type AgentSqlStarterId = AgentSqlTemplateStarterId | "database-question";

interface AgentContextualStartProps {
  page: AgentPageResource;
  title: string;
  reference?: AgentResourceReference;
  maskingRuleEditor?: boolean;
  loading: boolean;
  canStart: boolean;
  contextPending?: boolean;
  sqlEditor?: AgentSqlContextCapture;
  canCaptureSqlSelection?: boolean;
  sqlSuggestionsEnabled?: boolean;
  hasCurrentChat: boolean;
  draft: AgentDraft;
  onDraftChange: (draft: AgentDraft) => void;
  context?: string;
  contextTitle?: string;
  workspace?: { name: string; title: string };
  canMentionResources?: boolean;
  showAIConfigurationRecovery?: boolean;
  onStart: (
    prompt: string,
    reference?: AgentResourceReference,
    runMode?: AgentRunMode,
    references?: AgentResourceReference[],
    sqlStarter?: AgentSqlStarterId,
    sqlSuggestionSource?: AgentSqlSuggestionSource
  ) => void;
  onContinue: () => void;
}

export function AgentContextualStart({
  page,
  title,
  reference,
  maskingRuleEditor = false,
  loading,
  canStart,
  contextPending = false,
  sqlEditor,
  canCaptureSqlSelection = false,
  sqlSuggestionsEnabled = false,
  hasCurrentChat,
  draft,
  onDraftChange,
  context,
  contextTitle,
  workspace,
  canMentionResources = true,
  showAIConfigurationRecovery = false,
  onStart,
  onContinue,
}: AgentContextualStartProps) {
  const { t, i18n } = useTranslation();
  const userEmail = useAgentStore((state) => state.userEmail);
  const composerRef = useRef<HTMLDivElement>(null);
  const sendWithEnter = useAgentStore(
    (state) => state.preferences.sendWithEnter
  );
  const requiresReference =
    page.kind === "database" ||
    page.kind === "database-catalog" ||
    page.kind === "plan" ||
    page.kind === "issue" ||
    (page.kind === "plan-create" && Boolean(page.databaseName));
  const inputDisabled =
    !canStart || contextPending || showAIConfigurationRecovery;
  const resources = useAgentResourceDraft({
    draft,
    onDraftChange,
    enabled:
      !inputDisabled &&
      canMentionResources &&
      Boolean(workspace?.name) &&
      (!requiresReference || (!loading && Boolean(reference))),
    boundReferences: reference ? [reference] : [],
  });
  const canSend =
    canStart &&
    !contextPending &&
    !showAIConfigurationRecovery &&
    (!requiresReference || (!loading && Boolean(reference))) &&
    !resources.hasInvalidReferences &&
    !resources.hasUnboundMention &&
    Boolean(draft.text.trim());
  const sendPrompt = () => {
    if (!canSend) return;
    if (resources.references.length > 0) {
      onStart(draft.text.trim(), reference, "default", resources.references);
    } else {
      onStart(draft.text.trim(), reference, "default");
    }
  };
  const Icon =
    page.kind === "project"
      ? Folder
      : page.kind === "database"
        ? Database
        : page.kind === "database-catalog"
          ? Database
          : page.kind === "plan" || page.kind === "plan-create"
            ? ClipboardList
            : page.kind === "issue"
              ? CircleDot
              : page.kind === "global-masking"
                ? Shield
                : page.kind === "semantic-types"
                  ? Tags
                  : page.kind === "data-classification"
                    ? BadgeCheck
                    : page.kind === "masking-exemption"
                      ? Users
                      : Server;
  let heading = "";
  let suggestions: string[] = [];
  switch (page.kind) {
    case "project":
      heading = t("agent.contextual-start.project-heading");
      suggestions = [
        t("agent.landing-starters.project-next"),
        t("agent.landing-starters.project-open-blockers"),
        t("agent.landing-starters.project-open-recent"),
      ];
      break;
    case "database":
      heading = t("agent.contextual-start.database-heading");
      suggestions = [
        t("agent.contextual-start.database-schema"),
        t("agent.contextual-start.database-recent"),
      ];
      break;
    case "database-catalog":
      heading = t("agent.contextual-start.masking-catalog-heading");
      suggestions = [
        t("agent.contextual-start.masking-catalog-visibility"),
        t("agent.contextual-start.masking-catalog-rule"),
        t("agent.contextual-start.masking-catalog-exemption"),
      ];
      break;
    case "plan":
      heading = t("agent.contextual-start.plan-heading");
      suggestions = [
        t("agent.contextual-start.plan-changes"),
        t("agent.contextual-start.plan-risks"),
        t("agent.contextual-start.plan-status"),
      ];
      break;
    case "plan-create":
      heading = t("agent.contextual-start.plan-create-heading");
      suggestions = [t("agent.contextual-start.plan-create-sql")];
      break;
    case "issue":
      heading = t("agent.contextual-start.issue-heading");
      suggestions = [
        t("agent.contextual-start.issue-summary"),
        t("agent.contextual-start.issue-changes"),
        t("agent.contextual-start.issue-security"),
      ];
      break;
    case "instance":
      heading = t("agent.contextual-start.instance-heading");
      suggestions = [
        t("agent.contextual-start.instance-sync"),
        t("agent.contextual-start.instance-connection-permissions"),
      ];
      break;
    case "instance-create":
      heading = t("agent.contextual-start.instance-create-heading");
      suggestions = [
        t("agent.contextual-start.instance-create-details"),
        t("agent.contextual-start.instance-create-transport"),
        t("agent.contextual-start.instance-create-troubleshoot"),
      ];
      break;
    case "global-masking":
      heading = t("agent.contextual-start.masking-global-heading");
      suggestions = maskingRuleEditor
        ? [
            t("agent.contextual-start.masking-rule-configure"),
            t("agent.contextual-start.masking-rule-effect"),
            t("agent.contextual-start.masking-rule-review"),
          ]
        : [
            t("agent.contextual-start.masking-global-effect"),
            t("agent.contextual-start.masking-global-columns"),
            t("agent.contextual-start.masking-global-exemptions"),
          ];
      break;
    case "semantic-types":
      heading = t("agent.contextual-start.masking-semantic-heading");
      suggestions = [
        t("agent.contextual-start.masking-semantic-algorithm"),
        t("agent.contextual-start.masking-semantic-rules"),
        t("agent.contextual-start.masking-semantic-short-values"),
      ];
      break;
    case "data-classification":
      heading = t("agent.contextual-start.masking-classification-heading");
      suggestions = [
        t("agent.contextual-start.masking-classification-effect"),
        t("agent.contextual-start.masking-classification-rules"),
        t("agent.contextual-start.masking-classification-exemptions"),
      ];
      break;
    case "masking-exemption":
      heading = t("agent.contextual-start.masking-exemption-heading");
      suggestions = [
        t("agent.contextual-start.masking-exemption-effect"),
        t("agent.contextual-start.masking-exemption-scope"),
        t("agent.contextual-start.masking-exemption-review"),
      ];
      break;
  }

  const sql = sqlEditor?.kind === "ready" ? sqlEditor.snapshot : undefined;
  const selection = sql?.editorTarget?.selection;
  const sqlStarters: AgentSqlTemplateStarterId[] | undefined = sqlEditor
    ? canCaptureSqlSelection && selection && selection.end > selection.start
      ? ["explain-selection", "review-selection", "improve-selection"]
      : sql?.connection
        ? ["explore-database", "write-query", "table-relationships"]
        : sql?.project || sqlEditor.kind === "pending"
          ? ["write-query", "find-database", "fix-error"]
          : ["write-query", "fix-error"]
    : undefined;
  if (sqlEditor) heading = t("agent.contextual-start.sql-heading");
  const canSuggestFromDatabase = Boolean(
    sqlSuggestionsEnabled &&
      sql?.connection &&
      canStart &&
      !contextPending &&
      !showAIConfigurationRecovery &&
      sqlStarters?.[0] === "explore-database"
  );
  const generatedSuggestions = useSqlEditorSuggestions({
    enabled: canSuggestFromDatabase,
    workspaceName: workspace?.name,
    userEmail: userEmail ?? undefined,
    connection: sql?.connection,
    locale: i18n?.resolvedLanguage ?? i18n?.language ?? "en-US",
  });
  const suggestionsLoading =
    canSuggestFromDatabase && generatedSuggestions.status === "loading";
  const sqlPrompts: Record<AgentSqlTemplateStarterId, string> = {
    "write-query": t("agent.contextual-start.sql-write-query"),
    "find-database": t("agent.contextual-start.sql-find-database"),
    "fix-error": t("agent.contextual-start.sql-fix-error"),
    "explore-database": t("agent.contextual-start.sql-explore-database"),
    "table-relationships": t("agent.contextual-start.sql-table-relationships"),
    "explain-selection": t("agent.contextual-start.sql-explain-selection"),
    "review-selection": t("agent.contextual-start.sql-review-selection"),
    "improve-selection": t("agent.contextual-start.sql-improve-selection"),
  };
  const prompts: {
    text: string;
    starter?: AgentSqlStarterId;
    source?: AgentSqlSuggestionSource;
  }[] =
    canSuggestFromDatabase && generatedSuggestions.status === "ready"
      ? generatedSuggestions.questions.map((text) => ({
          text,
          starter: "database-question",
          source: generatedSuggestions.source,
        }))
      : suggestionsLoading
        ? []
        : sqlStarters
          ? sqlStarters.map((starter) => ({
              text: sqlPrompts[starter],
              starter,
            }))
          : suggestions.map((text) => ({ text }));

  return (
    <div
      className="flex min-h-0 flex-1 flex-col"
      data-agent-contextual-start={page.kind}
    >
      <div className="flex min-h-0 flex-1 overflow-y-auto p-4">
        <div className="m-auto w-full max-w-xl py-2">
          <div className="flex min-w-0 items-center gap-2 text-sm font-medium text-control-light">
            <Icon className="size-4 shrink-0" aria-hidden="true" />
            <span className="min-w-0 truncate">{title}</span>
          </div>
          <h2 className="mt-3 text-xl font-semibold text-main">{heading}</h2>
          {showAIConfigurationRecovery && (
            <div className="mt-4">
              <AgentAIConfigurationRecovery />
            </div>
          )}
          {requiresReference && !loading && !reference ? (
            <p className="mt-4 text-sm text-control-light">
              {t("agent.contextual-start.resource-unavailable")}
            </p>
          ) : (
            <>
              <div className="mt-4 text-xs font-medium text-control-light">
                {t("agent.empty-chat.suggestions")}
              </div>
              <div
                className="mt-2 divide-y divide-block-border border-y border-block-border"
                aria-busy={suggestionsLoading || undefined}
              >
                {suggestionsLoading && (
                  <div
                    role="status"
                    aria-label={t("common.loading")}
                    className="divide-y divide-block-border"
                  >
                    {[0, 1, 2].map((row) => (
                      <div
                        key={row}
                        aria-hidden="true"
                        data-agent-sql-suggestion-placeholder
                        className="flex h-11 items-center px-2"
                      >
                        <div className="flex w-full flex-col gap-1 motion-safe:animate-pulse">
                          <div className="h-3 w-4/5 rounded-xs bg-control-bg-hover" />
                          <div className="h-3 w-3/5 rounded-xs bg-control-bg-hover" />
                        </div>
                      </div>
                    ))}
                  </div>
                )}
                {requiresReference && (
                  <Button
                    appearance="secondary"
                    size="sm"
                    className="h-auto w-full min-w-0 justify-between rounded-xs py-1.5 text-left whitespace-normal hover:text-accent"
                    disabled={
                      !canStart || contextPending || loading || !reference
                    }
                    onClick={() => onStart("", reference)}
                    data-agent-contextual-mention
                  >
                    <span className="min-w-0 flex-1 break-words">
                      {loading ? (
                        <LoaderCircle
                          className="size-4 animate-spin"
                          aria-label={t("common.loading")}
                        />
                      ) : (
                        `@${reference?.label ?? title}`
                      )}
                    </span>
                    <ArrowUpRight
                      className="size-4 shrink-0"
                      aria-hidden="true"
                    />
                  </Button>
                )}
                {prompts.map(({ text, starter, source }) => (
                  <Button
                    key={source ? text : (starter ?? text)}
                    appearance="secondary"
                    size="sm"
                    className="h-auto w-full min-w-0 justify-between rounded-xs py-1.5 text-left whitespace-normal hover:text-accent"
                    disabled={
                      !canStart ||
                      contextPending ||
                      (sqlEditor && showAIConfigurationRecovery) ||
                      (requiresReference && !reference)
                    }
                    onClick={() =>
                      source
                        ? onStart(
                            text,
                            reference,
                            "default",
                            undefined,
                            "database-question",
                            source
                          )
                        : starter
                          ? onStart(
                              text,
                              reference,
                              "default",
                              undefined,
                              starter
                            )
                          : onStart(text, reference)
                    }
                    data-agent-contextual-suggestion
                    data-agent-sql-starter={starter}
                  >
                    <span className="min-w-0 flex-1 break-words">{text}</span>
                    <ArrowUpRight
                      className="size-4 shrink-0"
                      aria-hidden="true"
                    />
                  </Button>
                ))}
              </div>
            </>
          )}
          {!canStart && (
            <p className="mt-4 text-sm text-control-light">
              {t("agent.contextual-start.active-request")}
            </p>
          )}
          {contextPending && (
            <p role="status" className="mt-4 text-sm text-control-light">
              {t("common.loading")}
            </p>
          )}
          {hasCurrentChat && (
            <Button
              appearance="solid"
              size="sm"
              className="mt-4"
              onClick={onContinue}
              disabled={contextPending}
              data-agent-contextual-continue
            >
              {t("agent.contextual-start.continue-chat")}
            </Button>
          )}
        </div>
      </div>
      <div
        ref={composerRef}
        className="shrink-0 border-t border-block-border p-3"
        data-agent-contextual-input
      >
        <AgentChatbox
          containerRef={composerRef}
          placeholder={`${t("agent.input-placeholder")} · ${sendWithEnter ? t("agent.input-keyboard-hint") : t("agent.input-keyboard-hint-newline")}`}
          footer={
            <AgentChatboxFooter
              context={context}
              contextTitle={contextTitle}
              workspace={workspace}
              contextDisabled
              contextReadOnly
              onSelectContext={() => false}
              chat={null}
              showEmptyUsage
            />
          }
          onSend={sendPrompt}
          sendDisabled={!canSend}
          textareaProps={{
            textareaRef: resources.textareaRef,
            value: draft.text,
            mentions: resources.mentions.map(({ resource, range }) => ({
              range,
              label: resource.label,
            })),
            onDeleteMention: resources.onDeleteMention,
            "aria-label": t("agent.input-placeholder"),
            disabled: inputDisabled,
            onChange: resources.onChange,
            onKeyDown: (event) => {
              if (resources.handleKeyDown(event)) return;
              if (isAgentSendShortcut(event.nativeEvent, sendWithEnter)) {
                event.preventDefault();
                sendPrompt();
              }
            },
          }}
        >
          {canMentionResources && workspace?.name && (
            <AgentResourcePicker
              {...resources.pickerProps}
              workspaceName={workspace.name}
              projectName={
                context?.startsWith(projectNamePrefix) ? context : undefined
              }
            />
          )}
        </AgentChatbox>
      </div>
    </div>
  );
}
