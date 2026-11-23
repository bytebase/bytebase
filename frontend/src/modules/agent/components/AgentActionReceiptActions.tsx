import { isEqual } from "lodash-es";
import { Loader2, RefreshCw } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_ISSUE_DETAIL } from "@/app/router/handles";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useTimeReading } from "@/hooks/useTimeReading";
import { cn } from "@/lib/utils";
import { useAppStore } from "@/stores/app";
import {
  getAccessGrantDisplayStatusText,
  getAccessGrantStatusTagType,
} from "@/utils/accessGrant";
import { formatAbsoluteDateTime } from "@/utils/datetime";
import {
  actionReceiptKey,
  actionReceiptStatusReading,
  decodeAccessGrantActionReceipt,
  isActionObservationNewer,
  normalizeActionReceipts,
} from "../logic/actionReceipt";
import { checkActionReceiptStatus } from "../logic/actionReceiptStatus";
import { resolveAgentHandoffRoute } from "../logic/handoff";
import {
  captureAgentPageHandoffContext,
  minimizeAgentAfterPageHandoff,
} from "../logic/pageHandoff";
import type {
  AgentActionObservation,
  AgentActionReceipt,
} from "../logic/types";
import { useAgentStore } from "../store/agent";
import { AgentPrimaryAction } from "./AgentPrimaryAction";

interface Props {
  chatId: string;
  messageId: string;
  receipt: AgentActionReceipt;
}

export function AgentActionReceiptActions({
  chatId,
  messageId,
  receipt,
}: Props) {
  const { t } = useTranslation();
  const workspaceName = useAgentStore((state) => state.workspaceName);
  const userEmail = useAgentStore((state) => state.userEmail);
  const message = useAgentStore((state) =>
    state.messagesByChatId[chatId]?.find(({ id }) => id === messageId)
  );
  const chat = useAgentStore((state) =>
    state.chats.find(({ id }) => id === chatId)
  );
  const currentUser = useAppStore((state) => state.currentUser);
  const workspace = useAppStore((state) => state.workspace);
  const owner = useRef({ workspaceName, userEmail }).current;
  const mounted = useRef(true);
  const checkingRef = useRef(false);
  const [checking, setChecking] = useState(false);
  const [opening, setOpening] = useState(false);
  const [openFailed, setOpenFailed] = useState(false);
  const [observation, setObservation] = useState(receipt.observation);
  const liveReceipt = {
    ...receipt,
    observation:
      observation && isActionObservationNewer(observation, receipt.observation)
        ? observation
        : receipt.observation,
  };
  const status = useTimeReading(actionReceiptStatusReading, liveReceipt);
  const isGrant = receipt.kind === "access-grant";
  const grant = isGrant ? decodeAccessGrantActionReceipt(receipt) : undefined;
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);

  const ownsSource = () => {
    const state = useAgentStore.getState(),
      app = useAppStore.getState();
    const sourceChat = state.getChat(chatId);
    const source = state.messagesByChatId[chatId]?.find(
      ({ id }) => id === messageId
    );
    const { observation: _observation, ...saved } = receipt;
    return (
      mounted.current &&
      !!owner.workspaceName &&
      !!owner.userEmail &&
      state.workspaceName === owner.workspaceName &&
      state.userEmail === owner.userEmail &&
      app.workspace?.name === owner.workspaceName &&
      app.currentUser?.workspace === owner.workspaceName &&
      app.currentUser.email === owner.userEmail &&
      (!grant || grant.creator === `users/${owner.userEmail}`) &&
      sourceChat?.workspaceName === owner.workspaceName &&
      !sourceChat.archived &&
      normalizeActionReceipts(
        source?.metadata?.actionReceipts,
        source?.metadata,
        owner.userEmail
      )?.some((candidate) => {
        const { observation: _candidateObservation, ...candidateSave } =
          candidate;
        return isEqual(candidateSave, saved);
      }) === true
    );
  };
  const checkStatus = async () => {
    if (checkingRef.current || !ownsSource()) return;
    const scopeCurrent = useAgentStore.getState().captureApiScope();
    const isCurrent = () => scopeCurrent() && ownsSource();
    checkingRef.current = true;
    setChecking(true);
    setOpenFailed(false);
    const record = (result: AgentActionObservation) => {
      const updated = useAgentStore
        .getState()
        .recordActionObservation(
          chatId,
          messageId,
          actionReceiptKey(receipt),
          result,
          { workspaceName: owner.workspaceName!, userEmail: owner.userEmail! }
        );
      const recorded = updated?.metadata?.actionReceipts?.find(
        (candidate) => actionReceiptKey(candidate) === actionReceiptKey(receipt)
      )?.observation;
      if (updated) setObservation(recorded ?? result);
    };
    try {
      const result = await checkActionReceiptStatus(receipt, isCurrent);
      if (isCurrent()) record(result);
    } catch {
      if (isCurrent()) {
        const unavailable = {
          status: "UNAVAILABLE" as const,
          observedAt: Math.max(Date.now(), receipt.savedAt),
        };
        try {
          record(unavailable);
        } catch {
          setObservation(unavailable);
        }
      }
    } finally {
      checkingRef.current = false;
      if (mounted.current) setChecking(false);
    }
  };
  const openResource = async () => {
    if (opening || !ownsSource()) return;
    const scopeCurrent = useAgentStore.getState().captureApiScope();
    const isCurrent = () => scopeCurrent() && ownsSource();
    const context = captureAgentPageHandoffContext(chatId);
    setOpening(true);
    setOpenFailed(false);
    try {
      const target = grant?.issue
        ? {
            name: PROJECT_V1_ROUTE_ISSUE_DETAIL,
            params: {
              projectId: grant.name.split("/")[1]!,
              issueId: grant.issue.split("/")[3]!,
            },
          }
        : receipt.kind === "masking-rule"
          ? resolveAgentHandoffRoute({ kind: "open-global-masking" })
          : receipt.kind === "masking-exemption"
            ? resolveAgentHandoffRoute({
                kind: "open-project-masking-exemptions",
                project: receipt.policy.split("/").slice(0, 2).join("/"),
              })
            : undefined;
      if (!target || typeof target === "string" || !("name" in target))
        throw new Error("Saved resource unavailable");
      await router.push(target);
      if (!isCurrent()) return;
      const route = router.currentRoute.value;
      if (
        route.name !== target.name ||
        Object.entries(target.params ?? {}).some(
          ([key, value]) => route.params[key] !== value
        )
      )
        throw new Error("Saved resource unavailable");
      minimizeAgentAfterPageHandoff(context);
    } catch {
      if (isCurrent()) setOpenFailed(true);
    } finally {
      if (mounted.current) setOpening(false);
    }
  };
  if (
    !workspaceName ||
    !userEmail ||
    workspaceName !== owner.workspaceName ||
    userEmail !== owner.userEmail ||
    workspace?.name !== owner.workspaceName ||
    currentUser?.workspace !== owner.workspaceName ||
    currentUser.email !== owner.userEmail ||
    chat?.archived ||
    !message
  )
    return null;
  const CheckIcon = checking ? Loader2 : RefreshCw;
  const customStatus = {
    SAVED: t("agent.action-receipt.saved"),
    MATCHING: t("agent.action-receipt.matching"),
    CHANGED: t("agent.action-receipt.changed"),
    NOT_FOUND: t("agent.action-receipt.not-found"),
    UNAVAILABLE: t("agent.action-receipt.unavailable"),
  } as const;
  const statusText =
    status in customStatus
      ? customStatus[status as keyof typeof customStatus]
      : getAccessGrantDisplayStatusText(
          status as Parameters<typeof getAccessGrantDisplayStatusText>[0]
        );
  const tagType = isGrant
    ? getAccessGrantStatusTagType(
        status as Parameters<typeof getAccessGrantStatusTagType>[0]
      )
    : status === "SAVED" || status === "MATCHING"
      ? "success"
      : "warning";
  const title =
    receipt.kind === "masking-rule" &&
    typeof receipt.rule.condition === "object" &&
    receipt.rule.condition &&
    "title" in receipt.rule.condition
      ? String(receipt.rule.condition.title ?? "")
      : undefined;
  return (
    <div
      className="flex min-w-0 flex-col items-start gap-2"
      data-agent-action-receipt={actionReceiptKey(receipt)}
      data-agent-submitted-access-grant={grant?.name}
    >
      {title && <p className="text-xs font-medium wrap-anywhere">{title}</p>}
      {openFailed || status === "UNAVAILABLE" || status === "UNKNOWN" ? (
        <p role="alert" className="text-xs text-warning wrap-anywhere">
          {openFailed
            ? isGrant
              ? t("agent.submitted-access-grant.open-unavailable")
              : t("agent.action-receipt.open-unavailable")
            : isGrant
              ? t("agent.submitted-access-grant.status-unavailable")
              : t("agent.action-receipt.status-unavailable")}
        </p>
      ) : (
        <div
          className="flex max-w-full flex-wrap items-center gap-2"
          aria-live="polite"
        >
          <Badge
            variant={tagType === "error" ? "destructive" : tagType}
            data-agent-action-status={status}
            data-agent-access-grant-status={isGrant ? status : undefined}
          >
            {statusText}
          </Badge>
          <span className="text-xs text-control-placeholder wrap-anywhere">
            {t("agent.submitted-access-grant.observed-at", {
              time: formatAbsoluteDateTime(
                liveReceipt.observation?.observedAt ?? receipt.savedAt
              ),
            })}
          </span>
        </div>
      )}
      <div className="-ml-2 flex max-w-full flex-wrap items-center gap-2">
        <AgentPrimaryAction
          loading={opening}
          onClick={() => void openResource()}
          data-agent-action-open-access-grant={isGrant ? "" : undefined}
          data-agent-action-open-receipt
        >
          {isGrant
            ? t("agent.submitted-access-grant.open")
            : receipt.kind === "masking-rule"
              ? t("agent.action-receipt.open-rule")
              : t("agent.action-receipt.open-exemption")}
        </AgentPrimaryAction>
        <Button
          appearance="ghost"
          size="sm"
          className="max-w-full min-w-0 gap-1.5"
          disabled={checking}
          aria-busy={checking || undefined}
          onClick={() => void checkStatus()}
          data-agent-action-check-access-grant={isGrant ? "" : undefined}
          data-agent-action-check-receipt
        >
          <CheckIcon
            className={cn("size-4 shrink-0", checking && "animate-spin")}
            aria-hidden="true"
          />
          <span className="min-w-0 truncate">
            {t("agent.submitted-access-grant.check-status")}
          </span>
        </Button>
      </div>
    </div>
  );
}
