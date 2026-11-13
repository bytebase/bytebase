import { useId, useState } from "react";
import { useTranslation } from "react-i18next";
import { Alert } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import {
  FormControlGroup,
  FormControlRow,
  FormError,
  FormLabel,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { getRunTokenBudget } from "../logic/tokenBudget";
import { formatTokenLimit, parseTokenLimit } from "../logic/tokenLimit";
import type { AgentChat } from "../logic/types";
import {
  AgentPopover,
  AgentPopoverContent,
  AgentPopoverTitle,
  AgentPopoverTrigger,
} from "./ui/AgentPopover";

const formatter = new Intl.NumberFormat();

interface AgentTokenBudgetStatusProps {
  chat: AgentChat | null;
  onIncreaseLimit: (value: number) => Promise<boolean | void>;
  onResume: () => Promise<boolean | void> | boolean | void;
  onEndRun: () => Promise<void> | void;
  disabled?: boolean;
  remoteBusy?: boolean;
  canResume?: boolean;
  pendingAttemptId?: string;
}

export function AgentTokenBudgetStatus(props: AgentTokenBudgetStatusProps) {
  return (
    <TokenBudgetStatus
      key={`${props.chat?.id}-${props.chat?.runBudget?.runId}`}
      {...props}
    />
  );
}

function TokenBudgetStatus({
  chat,
  onIncreaseLimit,
  onResume,
  onEndRun,
  disabled = false,
  remoteBusy = false,
  canResume = true,
  pendingAttemptId,
}: AgentTokenBudgetStatusProps) {
  const { t } = useTranslation();
  const id = useId();
  const [open, setOpen] = useState(false);
  const [draft, setDraft] = useState("");
  const [pending, setPending] = useState(false);
  const [failed, setFailed] = useState(false);
  const projection = chat
    ? getRunTokenBudget(chat, pendingAttemptId)
    : undefined;
  if (!chat) return null;
  const budget = projection?.budget;
  const state = projection?.state ?? "normal";
  const hasUnreportedUsage = projection?.hasUnreportedUsage ?? false;
  if (!budget || remoteBusy) return null;
  const paused = chat.runCheckpoint?.suspension === "token-limit";
  const showRetry = paused && state !== "exhausted";
  const active = chat.status === "running" || chat.status === "awaiting_user";
  const showThreshold = !paused && active && state !== "normal";
  const showPartialUsage =
    hasUnreportedUsage && (active || chat.status === "error");
  if (!paused && !showThreshold && !showPartialUsage) return null;
  const minimum = Math.max(budget.limit, budget.totalTokensUsed);
  const minimumLimit = formatTokenLimit(
    minimum < Number.MAX_SAFE_INTEGER ? minimum + 1 : Number.MAX_SAFE_INTEGER
  );
  const roundedLimit = (BigInt(minimum) / 1000n + 1n) * 1000n;
  const defaultLimit =
    roundedLimit <= BigInt(Number.MAX_SAFE_INTEGER)
      ? formatTokenLimit(Number(roundedLimit))
      : minimumLimit;
  const parsed = parseTokenLimit(draft);
  const invalid = parsed === null || parsed <= minimum;
  const busy = disabled || pending;
  const canIncrease = (paused && state === "exhausted") || showThreshold;
  const increaseAndResume = paused && canResume;
  const title = paused
    ? showRetry
      ? t("agent.interrupted")
      : t("agent.token-budget.paused")
    : active && state === "exhausted"
      ? t("agent.token-budget.reached")
      : active && state === "warning"
        ? t("agent.token-budget.warning")
        : t("agent.token-budget.partial");
  const used =
    hasUnreportedUsage && budget
      ? t("agent.token-usage.at-least", {
          count: formatter.format(budget.totalTokensUsed),
        })
      : budget
        ? formatter.format(budget.totalTokensUsed)
        : undefined;

  const perform = async (
    action: () => Promise<boolean | void> | boolean | void
  ) => {
    if (busy) return;
    setPending(true);
    setFailed(false);
    try {
      const result = await action();
      if (result === false) setFailed(true);
      else setOpen(false);
    } catch {
      setFailed(true);
    } finally {
      setPending(false);
    }
  };

  return (
    <Alert
      appearance="caption"
      variant={(paused && !showRetry) || showThreshold ? "warning" : "info"}
      role="region"
      aria-label={t("agent.token-budget.title")}
      data-agent-token-budget
    >
      <div className="flex min-w-0 flex-col gap-2">
        <span role="status" aria-live="polite" aria-atomic="true">
          {title}
        </span>
        <div className="flex flex-wrap items-center justify-between gap-2">
          {budget && (
            <span className="tabular-nums">
              {t("agent.token-budget.usage", {
                used,
                limit: formatter.format(budget.limit),
              })}
            </span>
          )}
          <div className="flex flex-wrap items-center gap-2">
            {(canIncrease || open) && (
              <AgentPopover
                open={open}
                onOpenChange={(nextOpen) => {
                  if (pending) return;
                  setOpen(nextOpen);
                  setFailed(false);
                  if (nextOpen) setDraft(defaultLimit);
                }}
              >
                <AgentPopoverTrigger
                  render={
                    <Button
                      size="sm"
                      appearance={paused ? "solid" : "outline"}
                      disabled={busy}
                    />
                  }
                >
                  {t("agent.token-budget.increase")}
                </AgentPopoverTrigger>
                <AgentPopoverContent className="w-80 max-w-[calc(100vw-2rem)] p-4">
                  <div className="flex flex-col gap-4">
                    <AgentPopoverTitle className="font-medium">
                      {t("agent.token-budget.increase-title")}
                    </AgentPopoverTitle>
                    <FormControlGroup>
                      <FormLabel htmlFor={`${id}-limit`}>
                        {t("agent.token-budget.run-limit")}
                      </FormLabel>
                      <FormControlRow>
                        <Input
                          id={`${id}-limit`}
                          type="number"
                          inputMode="decimal"
                          min={minimumLimit}
                          max={formatTokenLimit(Number.MAX_SAFE_INTEGER)}
                          step="0.001"
                          required
                          size="sm"
                          className="min-w-0"
                          value={draft}
                          disabled={busy}
                          aria-invalid={invalid}
                          aria-describedby={
                            invalid ? `${id}-unit ${id}-error` : `${id}-unit`
                          }
                          onChange={(event) => {
                            setDraft(event.target.value);
                            setFailed(false);
                          }}
                        />
                        <span
                          id={`${id}-unit`}
                          className="shrink-0 text-sm text-control-light"
                        >
                          {t("agent.preferences.thousand-tokens")}
                        </span>
                      </FormControlRow>
                      {invalid && (
                        <FormError id={`${id}-error`}>
                          {t("agent.token-budget.invalid-limit", {
                            minimum: formatTokenLimit(minimum),
                          })}
                        </FormError>
                      )}
                    </FormControlGroup>
                    <div className="flex flex-wrap justify-end gap-2">
                      <Button
                        size="sm"
                        appearance="secondary"
                        disabled={pending}
                        onClick={() => setOpen(false)}
                      >
                        {t("common.cancel")}
                      </Button>
                      <Button
                        size="sm"
                        disabled={busy || invalid}
                        onClick={() => {
                          if (invalid || parsed === null) return;
                          void perform(async () => {
                            const saved = await onIncreaseLimit(parsed);
                            if (saved === false) return false;
                            setOpen(false);
                            if (increaseAndResume) return await onResume();
                          });
                        }}
                      >
                        {increaseAndResume
                          ? t("agent.token-budget.increase-and-resume")
                          : t("agent.token-budget.apply")}
                      </Button>
                    </div>
                    {failed && (
                      <FormError>
                        {t("agent.token-budget.action-failed")}
                      </FormError>
                    )}
                  </div>
                </AgentPopoverContent>
              </AgentPopover>
            )}
            {showRetry && (
              <Button
                size="sm"
                disabled={busy || !canResume}
                onClick={() => void perform(onResume)}
              >
                {t("agent.retry-last-chat-turn")}
              </Button>
            )}
            {paused && (
              <Button
                size="sm"
                appearance="secondary"
                disabled={busy}
                onClick={() => void perform(onEndRun)}
              >
                {t("agent.token-budget.end-run")}
              </Button>
            )}
          </div>
        </div>
        {hasUnreportedUsage && (paused || showThreshold) && (
          <span>{t("agent.token-budget.partial")}</span>
        )}
        {failed && !open && (
          <FormError>{t("agent.token-budget.action-failed")}</FormError>
        )}
      </div>
    </Alert>
  );
}
