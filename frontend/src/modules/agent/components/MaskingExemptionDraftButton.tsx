import { useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE } from "@/app/router/handles";
import { Alert } from "@/components/ui/alert";
import { useAppStore } from "@/stores/app";
import {
  consumeMaskingExemptionDraft,
  getMaskingExemptionDraft,
  getMaskingExemptionDraftSource,
  revalidateMaskingExemptionDraft,
} from "../logic/maskingExemptionDraft";
import {
  applyMaskingExemptionDraftInEditor,
  isMaskingExemptionDraftTargetReady,
  openMaskingExemptionDraftInEditor,
} from "../logic/maskingExemptionDraftTarget";
import type { MaskingRuleDraftContext } from "../logic/maskingRuleDraft";
import {
  captureAgentPageHandoffContext,
  minimizeAgentAfterPageHandoff,
} from "../logic/pageHandoff";
import { useAgentStore } from "../store/agent";
import { AgentPrimaryAction } from "./AgentPrimaryAction";

function assertCurrentContext(context: MaskingRuleDraftContext): void {
  const app = useAppStore.getState();
  const agent = useAgentStore.getState();
  if (
    app.currentUser?.name !== context.userName ||
    app.workspaceResourceName() !== context.workspaceName ||
    agent.currentChatId !== context.chatId ||
    agent.getChat(context.chatId)?.workspaceName !== context.workspaceName
  ) {
    throw new Error(
      "The masking exemption draft context is no longer current."
    );
  }
}

function assertCurrentTarget(targetPath: string): void {
  if (router.currentRoute.value.fullPath !== targetPath)
    throw new Error("The masking exemption editor route is no longer current.");
}

async function waitForTarget(
  context: MaskingRuleDraftContext,
  targetPath: string
): Promise<void> {
  for (let attempt = 0; attempt < 60; attempt++) {
    assertCurrentContext(context);
    assertCurrentTarget(targetPath);
    if (isMaskingExemptionDraftTargetReady(context)) return;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error("The masking exemption editor is not available.");
}

export function MaskingExemptionDraftButton({
  draftId,
  chatId,
  messageId,
}: {
  draftId: string;
  chatId: string;
  messageId: string;
}) {
  const { t } = useTranslation();
  const userName = useAppStore((state) => state.currentUser?.name);
  const workspaceName = useAppStore((state) => state.workspaceResourceName());
  const ownsSelectedChat = useAgentStore(
    (state) =>
      state.currentChatId === chatId &&
      state.getChat(chatId)?.workspaceName === workspaceName
  );
  const completed = useAgentStore((state) => {
    if (!ownsSelectedChat || !userName || !workspaceName) return undefined;
    const handoff = state
      .getMessages(chatId)
      .find((message) => message.id === messageId)
      ?.metadata?.maskingExemptionDraftHandoff;
    return handoff?.draftId === draftId
      ? handoff.created
        ? "created"
        : handoff.outcome
      : undefined;
  });
  const [loading, setLoading] = useState(false);
  const [unavailable, setUnavailable] = useState(false);
  const inFlight = useRef(false);
  const context =
    userName && workspaceName && ownsSelectedChat
      ? { workspaceName, userName, chatId }
      : undefined;
  const draft = context && getMaskingExemptionDraft(draftId, context);
  const source = context && getMaskingExemptionDraftSource(draftId, context);
  const label =
    completed === "created"
      ? t("common.created")
      : completed === "opened"
        ? t("agent.masking-exemption-draft.opened")
        : completed === "applied"
          ? t("agent.masking-exemption-draft.applied")
          : source
            ? t("agent.masking-exemption-draft.apply")
            : t("agent.masking-exemption-draft.open");

  const openDraft = async () => {
    if (inFlight.current || completed) return;
    const handoffContext = captureAgentPageHandoffContext(chatId);
    inFlight.current = true;
    setLoading(true);
    setUnavailable(false);
    try {
      if (!context || !getMaskingExemptionDraft(draftId, context))
        throw new Error("The masking exemption draft is unavailable.");
      assertCurrentContext(context);
      const expected = getMaskingExemptionDraftSource(draftId, context);
      const validated = await revalidateMaskingExemptionDraft(draftId, context);
      assertCurrentContext(context);
      const action = { draftId, messageId };
      if (expected) {
        await applyMaskingExemptionDraftInEditor(
          validated,
          expected,
          context,
          action
        );
      } else {
        const projectId = /^projects\/([^/]+)$/.exec(validated.project)?.[1];
        if (!projectId)
          throw new Error("The masking exemption project is unavailable.");
        const target = {
          name: PROJECT_V1_ROUTE_MASKING_EXEMPTION_CREATE,
          params: { projectId },
        };
        const targetPath = router.resolve(target).fullPath;
        if (targetPath !== `/projects/${projectId}/masking-exemption/create`)
          throw new Error("The masking exemption create route is unavailable.");
        await router.push(target);
        await waitForTarget(context, targetPath);
        assertCurrentContext(context);
        assertCurrentTarget(targetPath);
        await openMaskingExemptionDraftInEditor(validated, context, action);
      }
      assertCurrentContext(context);
      useAgentStore
        .getState()
        .recordMaskingExemptionDraftHandoff(
          chatId,
          messageId,
          draftId,
          expected ? "applied" : "opened"
        );
      consumeMaskingExemptionDraft(draftId, context);
      minimizeAgentAfterPageHandoff(handoffContext);
    } catch {
      setUnavailable(true);
    } finally {
      inFlight.current = false;
      setLoading(false);
    }
  };

  return (
    <div className="flex min-w-0 flex-col items-start gap-2">
      {draft && (
        <>
          <dl className="flex max-w-full min-w-0 flex-col gap-1 text-xs">
            <div className="flex flex-wrap items-baseline gap-x-2">
              <dt className="text-control-light">{t("common.members")}</dt>
              <dd className="min-w-0 wrap-anywhere">
                {draft.members.join(", ")}
              </dd>
            </div>
            <div className="flex flex-wrap items-baseline gap-x-2">
              <dt className="text-control-light">{t("common.resources")}</dt>
              <dd className="min-w-0 wrap-anywhere">
                {draft.scope.kind === "project"
                  ? draft.project
                  : draft.scope.database}
              </dd>
            </div>
            <div className="flex flex-wrap items-baseline gap-x-2">
              <dt className="text-control-light">{t("common.expiration")}</dt>
              <dd className="min-w-0 wrap-anywhere">
                {draft.expirationTimestamp ??
                  t("settings.sensitive-data.never-expires")}
              </dd>
            </div>
          </dl>
          <code className="max-h-24 max-w-full overflow-y-auto whitespace-pre-wrap text-xs wrap-anywhere">
            {draft.condition}
          </code>
        </>
      )}
      <AgentPrimaryAction
        className="-ml-2"
        data-agent-action-draft-masking-exemption
        actionKind={source ? "apply" : "open"}
        loading={loading}
        completed={Boolean(completed)}
        disabled={!draft}
        onClick={() => void openDraft()}
      >
        {label}
      </AgentPrimaryAction>
      {!completed && (
        <p className="text-xs text-control-light">
          {!draft && !unavailable
            ? t("agent.masking-exemption-draft.inactive")
            : t("agent.masking-exemption-draft.unsaved")}
        </p>
      )}
      {!completed && unavailable && (
        <Alert appearance="caption" variant="error">
          {t("agent.masking-exemption-draft.unavailable")}
        </Alert>
      )}
    </div>
  );
}
