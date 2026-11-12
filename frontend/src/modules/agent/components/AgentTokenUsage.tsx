import { ChevronDownIcon, XIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { getRunTokenBudget } from "../logic/tokenBudget";
import type { AgentChat, AgentModelCallUsage } from "../logic/types";
import {
  AgentPopover,
  AgentPopoverClose,
  AgentPopoverContent,
  AgentPopoverTitle,
  AgentPopoverTrigger,
} from "./ui/AgentPopover";

const formatter = new Intl.NumberFormat();
type Metric =
  | "totalTokens"
  | "inputTokens"
  | "outputTokens"
  | "cachedInputTokens";

export function AgentTokenUsage({
  chat,
  pendingAttemptId,
}: {
  chat: AgentChat | null;
  pendingAttemptId?: string;
}) {
  const { t } = useTranslation();
  if (!chat) return null;
  const calls = chat.lastRunUsage?.calls ?? [];
  const run = getRunTokenBudget(chat, pendingAttemptId);
  const format = (value: number | undefined) =>
    value === undefined
      ? t("agent.token-usage.not-reported")
      : formatter.format(value);
  const partial = (value: number) =>
    t("agent.token-usage.at-least", { count: formatter.format(value) });
  const sum = (metric: Metric) =>
    calls.length > 0 && calls.every((call) => call[metric] !== undefined)
      ? calls.reduce((total, call) => total + call[metric]!, 0)
      : undefined;
  const total = sum("totalTokens");
  const answerTotal =
    total !== undefined
      ? format(total)
      : calls.some((call) => call.totalTokens !== undefined)
        ? partial(
            calls.reduce((value, call) => value + (call.totalTokens ?? 0), 0)
          )
        : format(undefined);
  const chatTotal = chat.hasUnreportedUsage
    ? partial(chat.totalTokensUsed)
    : format(chat.totalTokensUsed);
  const callName = (call: AgentModelCallUsage) =>
    call.purpose === "follow-up"
      ? t("agent.token-usage.follow-up-questions")
      : call.toolNames.join(", ") || t("agent.token-usage.answer");

  return (
    <AgentPopover>
      <AgentPopoverTrigger
        render={
          <Button
            appearance="secondary"
            size="sm"
            className="group h-auto max-w-full text-control-light whitespace-normal"
          />
        }
      >
        {t("agent.chat-total-tokens", { count: chatTotal })}
        <ChevronDownIcon className="shrink-0 group-data-popup-open:rotate-180" />
      </AgentPopoverTrigger>
      <AgentPopoverContent
        className="w-112 max-w-[calc(100vw-2rem)] p-4"
        data-agent-token-usage
      >
        <div className="flex items-center justify-between gap-2 pb-2">
          <AgentPopoverTitle className="font-medium">
            {t("agent.token-usage.title")}
          </AgentPopoverTitle>
          <AgentPopoverClose
            render={
              <Button
                appearance="secondary"
                size="sm"
                aria-label={t("common.close")}
              />
            }
          >
            <XIcon />
          </AgentPopoverClose>
        </div>
        {run && (
          <div className="flex flex-col gap-2 pb-3" data-agent-run-usage>
            <div className="flex items-baseline justify-between gap-4">
              <span>{t("agent.token-usage.run-usage")}</span>
              <span className="font-medium tabular-nums">
                {run.hasUnreportedUsage
                  ? partial(run.budget.totalTokensUsed)
                  : format(run.budget.totalTokensUsed)}
              </span>
            </div>
            <div className="flex items-baseline justify-between gap-4">
              <span>{t("agent.token-usage.run-limit")}</span>
              <span className="tabular-nums">{format(run.budget.limit)}</span>
            </div>
            {run.failedAttemptCount > 0 && (
              <span className="text-xs text-control-light">
                {t("agent.token-usage.failed-attempts", {
                  count: run.failedAttemptCount,
                })}
              </span>
            )}
            {run.hasUnreportedUsage && (
              <span className="text-xs text-control-light">
                {t("agent.token-budget.partial")}
              </span>
            )}
          </div>
        )}
        <div className="flex items-baseline justify-between gap-4 pb-3">
          <span>{t("agent.token-usage.chat-total")}</span>
          <span className="font-medium tabular-nums">{chatTotal}</span>
        </div>
        <Separator />
        <div className="flex flex-wrap items-center justify-between gap-2 pt-3 pb-2">
          <span>{t("agent.token-usage.latest-answer")}</span>
          {calls.length > 0 && (
            <span className="text-xs text-control-light">
              {t("agent.token-usage.model-calls", { count: calls.length })}
            </span>
          )}
        </div>
        {calls.length === 0 ? (
          <div className="text-xs text-control-light">
            {t("agent.token-usage.not-recorded")}
          </div>
        ) : (
          <>
            <Table className="table-fixed text-xs [&_td]:wrap-anywhere [&_th]:wrap-anywhere [&_tr>:first-child]:pl-0 [&_tr>:last-child]:pr-0">
              <TableHeader>
                <TableRow>
                  <TableHead className="w-1/4 px-1 text-xs sm:w-2/5">
                    {t("agent.token-usage.model-call")}
                  </TableHead>
                  <TableHead className="px-1 text-right text-xs">
                    {t("agent.token-usage.input")}
                  </TableHead>
                  <TableHead className="px-1 text-right text-xs">
                    {t("agent.token-usage.output")}
                  </TableHead>
                  <TableHead className="px-1 text-right text-xs">
                    {t("agent.token-usage.total")}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody striped={false}>
                {calls.map((call, index) => (
                  <TableRow key={`${chat.lastRunUsage!.runId}-${index}`}>
                    <TableCell className="break-all px-1 py-2 text-xs">
                      {callName(call)}
                    </TableCell>
                    <TableCell className="px-1 py-2 text-right text-xs tabular-nums">
                      {format(call.inputTokens)}
                    </TableCell>
                    <TableCell className="px-1 py-2 text-right text-xs tabular-nums">
                      {format(call.outputTokens)}
                    </TableCell>
                    <TableCell className="px-1 py-2 text-right text-xs tabular-nums">
                      {format(call.totalTokens)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
              <tfoot>
                <TableRow>
                  <TableCell className="px-1 py-2 text-xs font-medium">
                    {t("agent.token-usage.answer-total")}
                  </TableCell>
                  <TableCell className="px-1 py-2 text-right text-xs tabular-nums">
                    {format(sum("inputTokens"))}
                  </TableCell>
                  <TableCell className="px-1 py-2 text-right text-xs tabular-nums">
                    {format(sum("outputTokens"))}
                  </TableCell>
                  <TableCell className="px-1 py-2 text-right text-xs tabular-nums">
                    {answerTotal}
                  </TableCell>
                </TableRow>
              </tfoot>
            </Table>
            <div
              className="flex justify-between gap-4 pt-3 text-xs text-control-light"
              data-agent-cached-input
            >
              <span>{t("agent.token-usage.cached-input")}</span>
              <span className="tabular-nums">
                {format(sum("cachedInputTokens"))}
              </span>
            </div>
          </>
        )}
      </AgentPopoverContent>
    </AgentPopover>
  );
}
