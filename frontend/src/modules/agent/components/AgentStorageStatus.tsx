import { RotateCw } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";

interface AgentStorageStatusProps {
  pressure?: {
    reason: "budget" | "quota" | "unavailable";
    phase: "admission" | "saving";
    bytes: number;
    limitBytes: number;
    otherScopeBytes?: number;
  };
  onRetrySaving: () => Promise<boolean>;
  disabled?: boolean;
}

export function AgentStorageStatus({
  pressure,
  onRetrySaving,
  disabled = false,
}: AgentStorageStatusProps) {
  const { t } = useTranslation();
  const [pending, setPending] = useState(false);
  const pendingRef = useRef(false);

  const retrySaving = async () => {
    if (pendingRef.current || disabled) return;
    pendingRef.current = true;
    setPending(true);
    try {
      await onRetrySaving();
    } catch {
      // Pressure remains visible until the parent confirms a durable save.
    } finally {
      pendingRef.current = false;
      setPending(false);
    }
  };

  if (!pressure) return null;

  return (
    <Alert
      appearance="caption"
      variant="warning"
      role="region"
      aria-label={t("agent.storage.title")}
      data-agent-storage-status
    >
      <div className="flex min-w-0 flex-wrap items-center gap-2">
        <p role="status" aria-live="polite" aria-atomic="true">
          {t("agent.storage.save-failed")}
        </p>
        <Button
          size="sm"
          appearance="outline"
          disabled={disabled || pending}
          aria-busy={pending}
          onClick={() => void retrySaving()}
        >
          <RotateCw aria-hidden="true" />
          {t("agent.storage.retry")}
        </Button>
      </div>
    </Alert>
  );
}
