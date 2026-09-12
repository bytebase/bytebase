import type { ReactNode } from "react";
import { MCPModeBadge } from "@/components/mcp/MCPModeBadge";
import type { MCPMode } from "@/components/mcp/mcpPolicy";

export interface MCPConsentLine {
  readonly key: string;
  readonly icon: ReactNode;
  /**
   * What the mark means, for a reader who cannot see it. Allowed and refused
   * are otherwise carried by a green check against a red cross — colour and
   * glyph, both visual — on the screen where someone decides whether to hand an
   * agent access.
   */
  readonly mark?: string;
  readonly text: string;
}

interface Props {
  readonly label: string;
  /** Omitted by the panel for a ceiling this build has no name for. */
  readonly mode?: MCPMode;
  readonly lines: readonly MCPConsentLine[];
}

/**
 * The panel each consent screen shows: the workspace's policy, and what it
 * means for the session being approved. Shared so the three screens cannot
 * drift, which they were doing while hand-assembling the same markup.
 */
export function MCPConsentPolicyCard({ label, mode, lines }: Props) {
  return (
    <div className="bg-control-bg rounded-sm p-4 flex flex-col gap-3">
      <div className="flex items-center justify-between gap-x-2">
        <p className="text-sm text-control-light">{label}</p>
        {mode !== undefined && <MCPModeBadge mode={mode} />}
      </div>
      <ul role="list" className="text-sm text-main flex flex-col gap-2">
        {lines.map((line) => (
          <li key={line.key} className="flex items-start gap-2">
            <span className="mt-0.5 shrink-0" aria-hidden="true">
              {line.icon}
            </span>
            {line.mark && <span className="sr-only">{line.mark}</span>}
            <span>{line.text}</span>
          </li>
        ))}
      </ul>
    </div>
  );
}
