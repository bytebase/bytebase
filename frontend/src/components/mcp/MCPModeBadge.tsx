import { useTranslation } from "react-i18next";
import {
  MCP_MODE_PRESENTATION,
  type MCPMode,
} from "@/components/mcp/mcpPolicy";
import { Badge } from "@/components/ui/badge";

interface Props {
  readonly mode: MCPMode;
  /**
   * A sentence naming what this badge reports, already translated. Where it is
   * given, the badge announces it instead of the bare mode name — for the
   * surface where the badge is the subject of the view and its name alone
   * would not say what it names.
   *
   * Resolved by the caller rather than threaded in as a locale key, so the key
   * stays a literal translation call the unused-key checker can trace.
   */
  readonly describedAs?: string;
}

/**
 * A mode's identity: its glyph, its name, and its color.
 *
 * One component rather than one per surface, because the identity has to be the
 * same on the settings page and on the consent page — the admin who picked the
 * eye is shown the eye when a session asks to connect — and a comment saying so
 * is not what keeps two hand-assembled badges in step.
 */
export function MCPModeBadge({ mode, describedAs }: Props) {
  const { t } = useTranslation();
  const { key, icon: Icon, badge } = MCP_MODE_PRESENTATION[mode];
  const label = t(`settings.mcp.policy.mode.${key}.title`);
  return (
    <Badge variant={badge} className="gap-x-1">
      <Icon className="size-3.5 shrink-0" aria-hidden="true" />
      {/* Carried as text rather than aria-label: a Badge renders a bare span,
          and ARIA forbids naming its implicit `generic` role, so a label put
          there is dropped. Swapping which copy each audience gets keeps the
          sentence whole in every locale's word order. */}
      {describedAs ? (
        <>
          <span className="sr-only">{describedAs}</span>
          <span aria-hidden="true">{label}</span>
        </>
      ) : (
        label
      )}
    </Badge>
  );
}
