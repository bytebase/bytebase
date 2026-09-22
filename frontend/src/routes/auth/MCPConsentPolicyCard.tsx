import type { ReactNode } from "react";
import { MCPModeBadge } from "@/components/mcp/MCPModeBadge";
import type { MCPMode } from "@/components/mcp/mcpPolicy";

export interface MCPConsentLine {
  readonly key: string;
  readonly icon: ReactNode;
  readonly text: string;
}

interface Props {
  readonly label: string;
  /** Omitted by the panel for a ceiling this build has no name for. */
  readonly mode?: MCPMode;
  readonly headerAction?: ReactNode;
  readonly modeAddon?: ReactNode;
  readonly lines?: readonly MCPConsentLine[];
  readonly children?: ReactNode;
}

/**
 * The panel each consent screen shows: the workspace's policy, and what it
 * means for the session being approved. Shared so the three screens cannot
 * drift apart.
 */
export function MCPConsentPolicyCard({
  label,
  mode,
  headerAction,
  modeAddon,
  lines,
  children,
}: Props) {
  return (
    <div className="bg-control-bg rounded-sm p-4 flex flex-col gap-3">
      <div className="flex flex-wrap items-center gap-x-2 gap-y-2">
        <div className="flex shrink-0 items-center gap-x-2">
          <p className="text-sm text-control-light">{label}</p>
          {headerAction}
        </div>
        {mode !== undefined && (
          <div className="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2 xl:ml-auto">
            <MCPModeBadge mode={mode} />
            {modeAddon}
          </div>
        )}
      </div>
      {children}
      {lines && (
        <ul role="list" className="text-sm text-main flex flex-col gap-2">
          {lines.map((line) => (
            <li key={line.key} className="flex items-start gap-2">
              <span className="mt-0.5 shrink-0" aria-hidden="true">
                {line.icon}
              </span>
              <span>{line.text}</span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
