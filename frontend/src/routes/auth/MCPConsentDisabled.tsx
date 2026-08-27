import { ScrollText, X } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

interface Props {
  readonly workspaceTitle: string;
  /** The workspace row, so a SaaS user can switch to one that permits MCP. */
  readonly workspaceCard: ReactNode;
  readonly onDismiss: () => void;
  readonly dismissing: boolean;
}

/**
 * The consent a disabled workspace never grants.
 *
 * This stops the person before they press Approve. Dismissing posts a denial
 * rather than walking history back, so the client waiting on the callback gets
 * access_denied instead of nothing; that path returns before the ceiling check
 * (backend/api/oauth2/authorize.go), so it writes no ceiling-refusal row. The
 * audit line here is about an attempt that reaches the grant path anyway,
 * which the backend refuses and records (backend/api/oauth2/consent_audit.go).
 */
export function MCPConsentDisabled({
  workspaceTitle,
  workspaceCard,
  onDismiss,
  dismissing,
}: Props) {
  const { t } = useTranslation();

  return (
    <div className="flex flex-col gap-6">
      <div className="text-center flex flex-col gap-2">
        <h1 className="text-xl font-semibold text-main">
          {t("oauth2.consent.mcp.disabled.title")}
        </h1>
        <p className="text-control">
          {/* The workspace load is fire-and-forget and swallows its error, so
              the title can be empty. Naming a workspace with nothing in the
              blank reads worse than not naming one. */}
          {workspaceTitle
            ? t("oauth2.consent.mcp.disabled.description", {
                workspace: workspaceTitle,
              })
            : t("oauth2.consent.mcp.disabled.description-no-workspace")}
        </p>
      </div>

      {workspaceCard}

      <div className="bg-control-bg rounded-sm p-4 flex flex-col gap-3">
        <div className="flex items-center justify-between gap-x-2">
          <p className="text-sm text-control-light">
            {t("oauth2.consent.mcp.disabled.policy-label")}
          </p>
          <Badge variant="destructive">
            {t("settings.mcp.policy.mode.disabled.title")}
          </Badge>
        </div>
        <ul className="text-sm text-main flex flex-col gap-2">
          <li className="flex items-start gap-2">
            <X className="mt-0.5 size-4 shrink-0 text-error" />
            <span>{t("oauth2.consent.mcp.disabled.line.no-session")}</span>
          </li>
          <li className="flex items-start gap-2">
            <ScrollText className="mt-0.5 size-4 shrink-0 text-control-light" />
            <span>{t("oauth2.consent.mcp.disabled.line.recorded")}</span>
          </li>
        </ul>
      </div>

      <div className="flex flex-col gap-2">
        <Button
          appearance="outline"
          size="lg"
          disabled={dismissing}
          onClick={onDismiss}
        >
          {t("common.close")}
        </Button>
        <p className="text-xs text-control-light text-center">
          {t("oauth2.consent.mcp.disabled.ask-admin")}
        </p>
      </div>
    </div>
  );
}
