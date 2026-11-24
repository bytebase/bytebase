import { RotateCw, Trash2 } from "lucide-react";
import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { AgentErrorDetails } from "./AgentErrorDetails";
import {
  AgentPopover,
  AgentPopoverContent,
  AgentPopoverTitle,
} from "./ui/AgentPopover";

interface AgentStorageStatusProps {
  pressure?: {
    reason: "budget" | "quota" | "unavailable";
    phase: "admission" | "saving";
    bytes: number;
    limitBytes: number;
    otherScopeBytes?: number;
  };
  onRetrySaving: () => Promise<boolean>;
  onFreeSpaceAndRetry?: () => Promise<boolean>;
  disabled?: boolean;
  error?: string | null;
}

export function AgentStorageStatus({
  pressure,
  onRetrySaving,
  onFreeSpaceAndRetry,
  disabled = false,
  error,
}: AgentStorageStatusProps) {
  const { t } = useTranslation();
  const [pending, setPending] = useState(false);
  const pendingRef = useRef(false);
  const [confirmationOpen, setConfirmationOpen] = useState(false);
  const [freeFailed, setFreeFailed] = useState(false);
  const recoveryRef = useRef<HTMLButtonElement>(null);

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

  const freeSpaceAndRetry = async () => {
    if (pendingRef.current || disabled || !onFreeSpaceAndRetry) return;
    pendingRef.current = true;
    setPending(true);
    setFreeFailed(false);
    try {
      if (await onFreeSpaceAndRetry()) {
        setConfirmationOpen(false);
      } else {
        setFreeFailed(true);
      }
    } catch {
      setFreeFailed(true);
    } finally {
      pendingRef.current = false;
      setPending(false);
    }
  };

  if (!pressure) return null;
  const full = pressure.reason !== "unavailable";

  return (
    <>
      <Alert
        variant="error"
        showIcon={false}
        className="rounded-sm border-0 bg-error/10 px-3 py-2 text-sm shadow-none"
        aria-label={t("agent.storage.title")}
        data-agent-storage-status
      >
        <p className="min-w-0 break-words" data-agent-storage-header>
          {full ? t("agent.storage.full") : t("agent.storage.save-failed")}
        </p>
        <AgentErrorDetails
          error={error}
          actions={
            <Button
              ref={recoveryRef}
              size="sm"
              appearance="outline"
              className="h-auto min-h-7 min-w-0 max-w-full gap-1.5 py-1 whitespace-normal wrap-anywhere"
              disabled={disabled || pending || (full && !onFreeSpaceAndRetry)}
              aria-busy={pending}
              aria-haspopup={full ? "dialog" : undefined}
              aria-expanded={full ? confirmationOpen : undefined}
              onClick={() => {
                if (!full) {
                  void retrySaving();
                  return;
                }
                setFreeFailed(false);
                setConfirmationOpen(true);
              }}
            >
              {full ? (
                <Trash2 size={16} className="shrink-0" aria-hidden="true" />
              ) : (
                <RotateCw size={16} className="shrink-0" aria-hidden="true" />
              )}
              {full
                ? t("agent.storage.free-and-retry")
                : t("agent.storage.retry")}
            </Button>
          }
        />
      </Alert>
      <AgentPopover
        open={confirmationOpen}
        onOpenChange={(open) => {
          if (!pendingRef.current) setConfirmationOpen(open);
        }}
      >
        <AgentPopoverContent
          anchor={recoveryRef}
          finalFocus={recoveryRef}
          className="w-64 p-3"
          data-agent-storage-confirmation
        >
          <AgentPopoverTitle className="mb-3 font-normal">
            {t("agent.storage.free-title")}
          </AgentPopoverTitle>
          {freeFailed && (
            <p role="status" className="mb-3 text-xs text-error">
              {t("agent.storage.free-failed")}
            </p>
          )}
          <div className="flex flex-wrap justify-end gap-2">
            <Button
              appearance="outline"
              size="sm"
              className="h-auto min-h-7 min-w-0 max-w-full py-1 whitespace-normal wrap-anywhere"
              disabled={pending}
              onClick={() => setConfirmationOpen(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              variant="destructive"
              size="sm"
              className="h-auto min-h-7 min-w-0 max-w-full py-1 whitespace-normal wrap-anywhere"
              disabled={disabled || pending || !onFreeSpaceAndRetry}
              aria-busy={pending}
              onClick={() => void freeSpaceAndRetry()}
            >
              {t("agent.storage.free-and-retry")}
            </Button>
          </div>
        </AgentPopoverContent>
      </AgentPopover>
    </>
  );
}
