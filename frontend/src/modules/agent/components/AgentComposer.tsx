import type { ReactNode } from "react";

interface AgentComposerProps {
  children: ReactNode;
  header?: ReactNode;
  footer: ReactNode;
  action: ReactNode;
}

export function AgentComposer({
  children,
  header,
  footer,
  action,
}: AgentComposerProps) {
  return (
    <div
      className="min-w-0 rounded-sm border border-accent/60 bg-background focus-within:border-accent focus-within:ring-1 focus-within:ring-accent"
      data-agent-composer
    >
      {header ? <div className="px-3 pt-2">{header}</div> : null}
      <div className="relative" data-agent-composer-body>
        {children}
      </div>
      <div
        className="flex flex-wrap items-end justify-between gap-2 px-3 pb-3"
        data-agent-composer-footer
      >
        <div className="flex min-w-0 flex-1 flex-wrap items-center gap-2 text-xs text-control-light">
          {footer}
        </div>
        <div className="shrink-0">{action}</div>
      </div>
    </div>
  );
}
