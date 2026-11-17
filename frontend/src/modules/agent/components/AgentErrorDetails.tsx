import { ChevronRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import {
  Collapsible,
  CollapsiblePanel,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

export function AgentErrorDetails({ error }: { error?: string | null }) {
  const { t } = useTranslation();
  if (!error) return null;
  return (
    <Collapsible className="mt-1">
      <CollapsibleTrigger
        className="w-fit text-xs underline decoration-dotted underline-offset-2"
        data-agent-error-details
      >
        <ChevronRight className="size-3" />
        {t("error-page.error-details")}
      </CollapsibleTrigger>
      <AgentErrorDetailsPanel error={error} />
    </Collapsible>
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
