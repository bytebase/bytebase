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
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import type { AgentPageResource } from "../logic/chatContext";
import type { AgentResourceReference } from "../logic/types";
import { AgentAIConfigurationRecovery } from "./AgentAIConfigurationRecovery";

interface AgentContextualStartProps {
  page: AgentPageResource;
  title: string;
  reference?: AgentResourceReference;
  loading: boolean;
  canStart: boolean;
  hasCurrentChat: boolean;
  showAIConfigurationRecovery?: boolean;
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
  showAIConfigurationRecovery = false,
  onStart,
  onContinue,
}: AgentContextualStartProps) {
  const { t } = useTranslation();
  const requiresReference =
    page.kind === "database" ||
    page.kind === "database-catalog" ||
    page.kind === "plan" ||
    page.kind === "issue" ||
    (page.kind === "plan-create" && Boolean(page.databaseName));
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
      suggestions = [
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
            <div className="mt-2 divide-y divide-block-border border-y border-block-border">
              {requiresReference && (
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
                  disabled={!canStart || (requiresReference && !reference)}
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
