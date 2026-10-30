import {
  ArrowUpRight,
  CircleDot,
  Database,
  Folder,
  LoaderCircle,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import type { AgentPageResource } from "../logic/chatContext";
import type { AgentResourceReference } from "../logic/types";

interface AgentContextualStartProps {
  page: AgentPageResource;
  title: string;
  reference?: AgentResourceReference;
  loading: boolean;
  canStart: boolean;
  hasCurrentChat: boolean;
  onStart: (prompt: string, reference?: AgentResourceReference) => void;
  onContinue: () => void;
}

export function AgentContextualStart({
  page,
  title,
  reference,
  loading,
  canStart,
  hasCurrentChat,
  onStart,
  onContinue,
}: AgentContextualStartProps) {
  const { t } = useTranslation();
  const Icon =
    page.kind === "project"
      ? Folder
      : page.kind === "database"
        ? Database
        : CircleDot;
  const heading =
    page.kind === "project"
      ? t("agent.contextual-start.project-heading")
      : page.kind === "database"
        ? t("agent.contextual-start.database-heading")
        : t("agent.contextual-start.issue-heading");
  const suggestions =
    page.kind === "project"
      ? [
          t("agent.landing-starters.project-next"),
          t("agent.landing-starters.project-open-blockers"),
          t("agent.landing-starters.project-open-recent"),
        ]
      : page.kind === "database"
        ? [
            t("agent.contextual-start.database-schema"),
            t("agent.contextual-start.database-recent"),
          ]
        : [
            t("agent.contextual-start.issue-summary"),
            t("agent.contextual-start.issue-changes"),
            t("agent.contextual-start.issue-security"),
          ];

  return (
    <div
      className="flex min-h-0 flex-1 overflow-y-auto p-4"
      data-agent-contextual-start={page.kind}
    >
      <div className="m-auto w-full max-w-xl py-2">
        <div className="flex min-w-0 items-center gap-2 text-sm font-medium text-control-light">
          <Icon className="size-4 shrink-0" aria-hidden="true" />
          <span className="min-w-0 truncate">{title}</span>
        </div>
        <h2 className="mt-3 text-xl font-semibold text-main">{heading}</h2>
        {page.kind !== "project" && !loading && !reference ? (
          <p className="mt-4 text-sm text-control-light">
            {t("agent.contextual-start.resource-unavailable")}
          </p>
        ) : (
          <>
            <div className="mt-4 text-xs font-medium text-control-light">
              {t("agent.empty-chat.suggestions")}
            </div>
            <div className="mt-2 divide-y divide-block-border border-y border-block-border">
              {page.kind !== "project" && (
                <Button
                  appearance="secondary"
                  size="md"
                  className="h-auto w-full min-w-0 justify-between rounded-xs px-3 py-2 text-left whitespace-normal hover:text-accent"
                  disabled={!canStart || loading || !reference}
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
                  <ArrowUpRight className="shrink-0" aria-hidden="true" />
                </Button>
              )}
              {suggestions.map((suggestion) => (
                <Button
                  key={suggestion}
                  appearance="secondary"
                  size="md"
                  className="h-auto w-full min-w-0 justify-between rounded-xs px-3 py-2 text-left whitespace-normal hover:text-accent"
                  disabled={
                    !canStart || (page.kind !== "project" && !reference)
                  }
                  onClick={() => onStart(suggestion, reference)}
                  data-agent-contextual-suggestion
                >
                  <span className="min-w-0 flex-1 break-words">
                    {suggestion}
                  </span>
                  <ArrowUpRight className="shrink-0" aria-hidden="true" />
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
        {hasCurrentChat && (
          <Button
            appearance="secondary"
            size="sm"
            className="mt-4"
            onClick={onContinue}
            data-agent-contextual-continue
          >
            {t("agent.contextual-start.continue-chat")}
          </Button>
        )}
      </div>
    </div>
  );
}
