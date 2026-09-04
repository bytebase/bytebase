import { ScrollText } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

/** Why this page cannot say what approving would grant. */
export type UndisclosedReason = "unknown" | "undisclosable";

interface Props {
  readonly reason: UndisclosedReason;
  /** The workspace row, so a SaaS user can switch to one that permits MCP. */
  readonly workspaceCard: ReactNode;
  /** Omitted where re-reading returns the same answer. See retryFor. */
  readonly onRetry?: () => void;
  readonly retrying: boolean;
  readonly onDismiss: () => void;
  readonly dismissing: boolean;
}

/**
 * The consent this page will not collect, because it cannot say what it is for.
 *
 * Every grant this server issues is an MCP grant and the workspace ceiling
 * decides what one is worth, so with no ceiling to disclose there is no Allow:
 * the disclosure is the consent (BOT-106).
 *
 * The button has to go rather than be left to fail. Of the two states, only
 * undisclosable also refuses the POST (backend/api/oauth2/consent_audit.go); a
 * failed read here says nothing about the read the POST makes for itself.
 */
export function MCPConsentUndisclosed({
  reason,
  workspaceCard,
  onRetry,
  retrying,
  onDismiss,
  dismissing,
}: Props) {
  const { t } = useTranslation();
  const key = `oauth2.consent.mcp.undisclosed.${reason}`;

  return (
    <div className="flex flex-col gap-6">
      <div className="text-center flex flex-col gap-2">
        <h1 className="text-xl font-semibold text-main">{t(`${key}.title`)}</h1>
        <p className="text-control">{t(`${key}.description`)}</p>
      </div>

      {workspaceCard}

      {/* The status, not a detail: this line is the whole reason there is no
          Allow, and it arrives after the policy read settles. Alert carries
          role="alert" so a screen reader announces it when it appears, which a
          styled div cannot do. Placed beside the details panel rather than
          inside it, the way MCPConsentCeiling places its write caution. */}
      <Alert variant="error" description={t(`${key}.line`)} />

      <div className="bg-control-bg rounded-sm p-4 flex flex-col gap-3">
        <p className="text-sm text-control-light">
          {t("oauth2.consent.mcp.undisclosed.policy-label")}
        </p>
        <ul className="text-sm text-main flex flex-col gap-2">
          <li className="flex items-start gap-2">
            <ScrollText className="mt-0.5 size-4 shrink-0 text-control-light" />
            <span>
              {t("oauth2.consent.mcp.undisclosed.line.nothing-approved")}
            </span>
          </li>
        </ul>
      </div>

      <div className="flex gap-x-2">
        <Button
          appearance="outline"
          size="lg"
          className="flex-1"
          disabled={dismissing || retrying}
          onClick={onDismiss}
        >
          {t("common.close")}
        </Button>
        {onRetry && (
          <Button
            size="lg"
            className="flex-1"
            disabled={dismissing || retrying}
            onClick={onRetry}
          >
            {t(`${key}.retry`)}
          </Button>
        )}
      </div>
    </div>
  );
}
