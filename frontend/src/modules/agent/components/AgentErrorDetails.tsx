import { ChevronRight } from "lucide-react";
import type { ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import { AgentActionGroup } from "./AgentActionGroup";

export function AgentErrorDetails({
  error,
  actions,
}: {
  error?: string | null;
  actions?: ReactNode;
}) {
  const { t } = useTranslation();
  if (!error && !actions) return null;
  const details = error && (
    <Collapsible className={cn(!actions && "mt-1")}>
      <CollapsibleTrigger
        className="min-w-0 w-fit gap-x-1 text-xs underline decoration-dotted underline-offset-2"
        data-agent-error-details
      >
        <ChevronRight className="size-3 shrink-0" />
        <span className="min-w-0 wrap-anywhere">
          {t("error-page.error-details")}
        </span>
      </CollapsibleTrigger>
      <AgentErrorDetailsPanel error={error} />
    </Collapsible>
  );
  return actions ? (
    <div className="mt-2 flex min-w-0 flex-col gap-2" data-agent-error-footer>
      {details}
      <AgentActionGroup data-agent-error-actions>{actions}</AgentActionGroup>
    </div>
  ) : (
    details
  );
}

export function AgentErrorDetailsPanel({ error }: { error: string }) {
  return (
    <CollapsiblePanel className="pt-1">
      <pre className="max-h-40 overflow-auto whitespace-pre-wrap break-words font-mono text-xs">
        {error}
      </pre>
    </CollapsiblePanel>
  );
}
