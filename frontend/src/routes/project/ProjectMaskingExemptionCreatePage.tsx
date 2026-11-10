import { create } from "@bufbuild/protobuf";
import dayjs from "dayjs";
import { isEqual } from "lodash-es";
import { CircleHelp } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { flushSync } from "react-dom";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { router } from "@/app/router";
import { PROJECT_V1_ROUTE_MASKING_EXEMPTION } from "@/app/router/handles";
import { AccountMultiSelect } from "@/components/AccountMultiSelect";
import { DatabaseResourceSelector } from "@/components/DatabaseResourceSelector";
import { ExprEditor } from "@/components/ExprEditor";
import { FeatureAttention } from "@/components/FeatureAttention";
import { FeatureBadge } from "@/components/FeatureBadge";
import { Button } from "@/components/ui/button";
import { ExpirationPicker } from "@/components/ui/expiration-picker";
import { FeatureModal } from "@/components/ui/feature-modal";
import { FormControlGroup, FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { Tooltip } from "@/components/ui/tooltip";
import { useProjectByName } from "@/hooks/useProjectByName";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import {
  buildMaskingExemption,
  useMaskingExemptionExprConfig,
} from "@/lib/sensitive-data/maskingExemption";
import { AgentPageHelpButton } from "@/modules/agent/components/AgentPageHelpButton";
import {
  type MaskingExemptionDraft,
  parseMaskingExemptionDraftCondition,
  validateMaskingExemptionDraft,
} from "@/modules/agent/logic/maskingExemptionDraft";
import {
  type MaskingExemptionDraftOrigin,
  type MaskingExemptionEditorSnapshot,
  registerMaskingExemptionDraftTarget,
} from "@/modules/agent/logic/maskingExemptionDraftTarget";
import { useAgentStore } from "@/modules/agent/store/agent";
import type { ConditionGroupExpr } from "@/modules/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/modules/cel";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { projectNamePrefix } from "@/stores/modules/v1/common";
import type { DatabaseResource } from "@/types";
import {
  MaskingExemptionPolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import { Setting_SettingName } from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { batchConvertParsedExprToCELString } from "@/utils/v1/cel";

type RadioValue = "ALL" | "EXPRESSION" | "SELECT";

export function ProjectMaskingExemptionCreatePage({
  projectId,
}: {
  projectId: string;
}) {
  const { t } = useTranslation();
  const projectsByName = useAppStore((s) => s.projectsByName);
  const workspaceResourceName = useAppStore((s) => s.workspaceResourceName());
  const userName = useAppStore((s) => s.currentUser?.name);

  const projectName = `${projectNamePrefix}${projectId}`;
  const accountParents = useMemo(
    () => [...new Set([workspaceResourceName, projectName].filter(Boolean))],
    [workspaceResourceName, projectName]
  );
  // subscribe to re-render on project cache change
  void projectsByName;
  const project = useProjectByName(projectName);

  // Ensure classification config is loaded
  useEffect(() => {
    useAppStore
      .getState()
      .getOrFetchSettingByName(Setting_SettingName.DATA_CLASSIFICATION, true);
  }, []);

  const hasRequiredFeature = useAppStore((s) =>
    s.hasFeature(PlanFeature.FEATURE_DATA_MASKING)
  );

  // Form state
  const [radioValue, setRadioValue] = useState<RadioValue>("ALL");
  const [databaseResources, setDatabaseResources] = useState<
    DatabaseResource[]
  >([]);
  const [expr, setExpr] = useState<ConditionGroupExpr>(
    wrapAsGroup(emptySimpleExpr())
  );
  const [description, setDescription] = useState("");
  const [expirationTimestamp, setExpirationTimestamp] = useState<
    string | undefined
  >();
  const [memberList, setMemberList] = useState<string[]>([]);
  const [processing, setProcessing] = useState(false);
  const [savedSignature, setSavedSignature] = useState<string>();
  const [showFeatureModal, setShowFeatureModal] = useState(false);
  const [draftPreparing, setDraftPreparing] = useState(false);
  const [chatId, setChatId] = useState<string>();
  const [sessionId] = useState(() => uuidv4());
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const signature = JSON.stringify([
    projectName,
    radioValue,
    databaseResources,
    expr,
    description,
    expirationTimestamp,
    memberList,
  ]);
  const revision = useRef({ signature, value: 0 });
  if (revision.current.signature !== signature)
    revision.current = { signature, value: revision.current.value + 1 };
  const currentFields = useRef({
    projectName,
    radioValue,
    databaseResources,
    expr,
    description,
    expirationTimestamp,
    memberList,
    processing,
    chatId,
    revision: String(revision.current.value),
  });
  currentFields.current = {
    projectName,
    radioValue,
    databaseResources,
    expr,
    description,
    expirationTimestamp,
    memberList,
    processing,
    chatId,
    revision: String(revision.current.value),
  };
  const draftOperation = useRef(false);
  const draftOrigin = useRef<MaskingExemptionDraftOrigin | undefined>(
    undefined
  );
  const draftCallbacks = useRef({
    getSnapshot: async (): Promise<
      MaskingExemptionEditorSnapshot | undefined
    > => undefined,
    apply: async (
      _draft: MaskingExemptionDraft,
      _expected: MaskingExemptionEditorSnapshot | undefined,
      _chatId: string,
      _active: () => boolean,
      _origin?: MaskingExemptionDraftOrigin
    ) => {},
  });
  useUnsavedChangesGuard(Boolean(chatId) && signature !== savedSignature);

  const assertAvailable = (
    originatingChatId: string,
    active: () => boolean
  ) => {
    const app = useAppStore.getState();
    const agent = useAgentStore.getState();
    const chat = agent.getChat(originatingChatId);
    const fields = currentFields.current;
    if (
      !active() ||
      fields.processing ||
      fields.projectName !== projectName ||
      app.currentUser?.name !== userName ||
      app.workspaceResourceName() !== workspaceResourceName ||
      agent.currentChatId !== originatingChatId ||
      !chat ||
      chat.archived ||
      chat.workspaceName !== workspaceResourceName ||
      (fields.chatId && fields.chatId !== originatingChatId)
    )
      throw new Error(
        "The masking exemption editor is unavailable or belongs to another chat."
      );
  };
  draftCallbacks.current = {
    getSnapshot: async () => {
      const fields = currentFields.current;
      if (fields.processing) return undefined;
      let condition = "";
      let incomplete = false;
      if (fields.radioValue === "EXPRESSION") {
        incomplete = !validateSimpleExpr(fields.expr);
        if (!incomplete) {
          try {
            const parsed = await buildCELExpr(fields.expr);
            if (parsed)
              [condition] = await batchConvertParsedExprToCELString([parsed]);
            incomplete = !condition;
          } catch {
            incomplete = true;
          }
        }
      }
      if (
        fields.revision !== currentFields.current.revision ||
        currentFields.current.processing
      )
        return undefined;
      return {
        sessionId,
        revision: fields.revision,
        project: fields.projectName,
        chatId: fields.chatId,
        radioValue: fields.radioValue,
        databaseResources: structuredClone(fields.databaseResources),
        condition: condition ?? "",
        ...(incomplete ? { conditionStatus: "incomplete" as const } : {}),
        members: [...fields.memberList],
        description: fields.description,
        expirationTimestamp: fields.expirationTimestamp ?? null,
      };
    },
    apply: async (draft, expected, originatingChatId, active, origin) => {
      assertAvailable(originatingChatId, active);
      if (
        draftOperation.current ||
        draft.project !== currentFields.current.projectName
      )
        throw new Error(
          "The masking exemption editor is busy or belongs to another project."
        );
      const before = currentFields.current;
      if (expected) {
        if (
          expected.sessionId !== sessionId ||
          expected.revision !== before.revision ||
          expected.project !== before.projectName ||
          expected.chatId !== before.chatId ||
          !isEqual(expected, await draftCallbacks.current.getSnapshot())
        )
          throw new Error(
            "The masking exemption draft has changed. Prepare it again."
          );
      } else if (
        before.radioValue !== "ALL" ||
        before.memberList.length ||
        before.description ||
        before.expirationTimestamp ||
        before.databaseResources.length ||
        !isEqual(before.expr, wrapAsGroup(emptySimpleExpr()))
      )
        throw new Error(
          "The masking exemption form has unsaved edits. Prepare a draft for the current form."
        );
      draftOperation.current = true;
      setDraftPreparing(true);
      try {
        if (!(await useAppStore.getState().refreshSubscription()))
          throw new Error("The current subscription could not be verified.");
        const verified = await validateMaskingExemptionDraft(draft, {
          workspaceName: workspaceResourceName,
          userName: userName!,
          chatId: originatingChatId,
        });
        const nextMode: RadioValue =
          verified.scope.kind === "project" && verified.condition === "true"
            ? "ALL"
            : "EXPRESSION";
        const nextExpr =
          nextMode === "ALL"
            ? wrapAsGroup(emptySimpleExpr())
            : await parseMaskingExemptionDraftCondition(verified.condition);
        assertAvailable(originatingChatId, active);
        if (
          before.revision !== currentFields.current.revision ||
          before.chatId !== currentFields.current.chatId
        )
          throw new Error(
            "The masking exemption form has changed. Prepare it again."
          );
        setRadioValue(nextMode);
        setDatabaseResources([]);
        setExpr(nextExpr);
        setMemberList([...verified.members]);
        setDescription(verified.description);
        setExpirationTimestamp(verified.expirationTimestamp ?? undefined);
        setChatId(originatingChatId);
        draftOrigin.current = origin && { ...origin };
        const nextSignature = JSON.stringify([
          projectName,
          nextMode,
          [],
          nextExpr,
          verified.description,
          verified.expirationTimestamp ?? undefined,
          verified.members,
        ]);
        revision.current = {
          signature: nextSignature,
          value: revision.current.value + 1,
        };
        currentFields.current = {
          ...before,
          radioValue: nextMode,
          databaseResources: [],
          expr: nextExpr,
          memberList: [...verified.members],
          description: verified.description,
          expirationTimestamp: verified.expirationTimestamp ?? undefined,
          chatId: originatingChatId,
          revision: String(revision.current.value),
        };
      } finally {
        draftOperation.current = false;
        setDraftPreparing(false);
      }
    },
  };
  useEffect(() => {
    if (!workspaceResourceName || !userName) return;
    let active = true;
    const unregister = registerMaskingExemptionDraftTarget(
      { workspaceName: workspaceResourceName, userName },
      {
        getSnapshot: () => draftCallbacks.current.getSnapshot(),
        open: (draft, chatId, origin) =>
          draftCallbacks.current.apply(
            draft,
            undefined,
            chatId,
            () => active,
            origin
          ),
        apply: (draft, expected, chatId, origin) =>
          draftCallbacks.current.apply(
            draft,
            expected,
            chatId,
            () => active,
            origin
          ),
      }
    );
    return () => {
      active = false;
      unregister();
    };
  }, [workspaceResourceName, userName, projectName]);

  // Validation
  const isValid = useMemo(() => {
    switch (radioValue) {
      case "SELECT":
        return databaseResources.length > 0;
      case "EXPRESSION":
        return validateSimpleExpr(expr);
      default:
        return true;
    }
  }, [radioValue, databaseResources, expr]);

  const submitDisabled = useMemo(
    () => memberList.length === 0 || !isValid || signature === savedSignature,
    [memberList, isValid, signature, savedSignature]
  );

  const { factorList, factorOperatorOverrideMap, factorOptionConfigMap } =
    useMaskingExemptionExprConfig(projectName);

  const onRadioChange = useCallback(
    (value: RadioValue) => {
      if (!hasRequiredFeature && value !== "ALL") {
        setShowFeatureModal(true);
        return;
      }
      setRadioValue(value);
    },
    [hasRequiredFeature]
  );

  const onDismiss = useCallback(() => {
    router.back();
  }, []);

  const onSubmit = useCallback(async () => {
    if (processing || draftOperation.current || submitDisabled) return;
    const submittedOrigin = draftOrigin.current;
    setProcessing(true);

    try {
      const exemption = await buildMaskingExemption({
        radioValue,
        expr,
        databaseResources,
        memberList,
        description,
        expirationTimestamp,
      });

      const policy = await useAppStore
        .getState()
        .getOrFetchPolicyByParentAndType({
          parentPath: projectName,
          policyType: PolicyType.MASKING_EXEMPTION,
          refresh: true,
        });
      const existed =
        policy?.policy?.case === "maskingExemptionPolicy"
          ? policy.policy.value.exemptions
          : [];

      await useAppStore.getState().upsertPolicy({
        parentPath: projectName,
        policy: {
          name: policy?.name,
          type: PolicyType.MASKING_EXEMPTION,
          resourceType: PolicyResourceType.PROJECT,
          policy: {
            case: "maskingExemptionPolicy",
            value: create(MaskingExemptionPolicySchema, {
              exemptions: [...existed, exemption],
            }),
          },
        },
      });

      if (mounted.current) {
        // Commit the saved baseline before navigation checks the dirty guard.
        flushSync(() => setSavedSignature(signature));
      }
      const app = useAppStore.getState();
      if (
        submittedOrigin &&
        app.currentUser?.name === submittedOrigin.userName &&
        app.workspaceResourceName() === submittedOrigin.workspaceName
      ) {
        try {
          useAgentStore
            .getState()
            .markMaskingExemptionDraftCreated(
              submittedOrigin.chatId,
              submittedOrigin.messageId,
              submittedOrigin.draftId
            );
        } catch {
          // Local chat persistence cannot turn a saved exemption into a failed save.
        }
      }
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: t("common.created"),
      });
      if (
        mounted.current &&
        revision.current.signature === signature &&
        app.currentUser?.name === userName &&
        app.workspaceResourceName() === workspaceResourceName
      ) {
        await router.push({
          name: PROJECT_V1_ROUTE_MASKING_EXEMPTION,
          params: { projectId },
        });
      }
    } catch (error) {
      pushNotification({
        module: "bytebase",
        style: "CRITICAL",
        title: `${error}`,
      });
    } finally {
      setProcessing(false);
    }
  }, [
    processing,
    expirationTimestamp,
    radioValue,
    expr,
    databaseResources,
    memberList,
    description,
    projectName,
    projectId,
    userName,
    workspaceResourceName,
    t,
    signature,
    submitDisabled,
  ]);

  // Min datetime for expiration picker (today start of day)
  const minDatetime = useMemo(() => {
    return dayjs().startOf("day").format("YYYY-MM-DDTHH:mm");
  }, []);

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="px-4 pt-4">
        <div className="flex flex-wrap items-center gap-2">
          <h2 className="text-lg font-medium">
            {t("project.masking-exemption.grant-exemption")}
          </h2>
          <AgentPageHelpButton />
        </div>
        <div className="border-b border-block-border mt-3" />
      </div>

      {/* Body */}
      <div className="flex-1 mb-6 px-4 overflow-y-auto">
        <div className="flex flex-col gap-y-8 pt-4">
          <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />

          {/* Resources */}
          <div className="w-full">
            <div className="flex items-center gap-x-1 mb-2">
              <span className="text-main">{t("common.resources")}</span>
              <span className="text-error">*</span>
            </div>

            {/* Radio group */}
            <div className="w-full mb-2">
              <RadioGroup
                className="flex-col items-start gap-2 sm:flex-row sm:items-center sm:gap-4"
                value={radioValue}
                onValueChange={(value) => onRadioChange(value as RadioValue)}
              >
                <Tooltip content={t("issue.role-grant.all-databases-tip")}>
                  <RadioGroupItem value="ALL">
                    {t("issue.role-grant.all-databases")}
                  </RadioGroupItem>
                </Tooltip>

                <RadioGroupItem value="EXPRESSION" disabled={!project}>
                  <div className="flex items-center gap-x-1">
                    <FeatureBadge
                      feature={PlanFeature.FEATURE_DATA_MASKING}
                      clickable={false}
                    />
                    <span>{t("issue.role-grant.use-cel")}</span>
                  </div>
                </RadioGroupItem>

                <RadioGroupItem value="SELECT" disabled={!project}>
                  <div className="flex items-center gap-x-1">
                    <FeatureBadge
                      feature={PlanFeature.FEATURE_DATA_MASKING}
                      clickable={false}
                    />
                    <span>{t("issue.role-grant.manually-select")}</span>
                  </div>
                </RadioGroupItem>
              </RadioGroup>
            </div>

            {/* Resource selector content */}
            {radioValue === "SELECT" && (
              <DatabaseResourceSelector
                projectName={projectName}
                value={databaseResources}
                includeColumns
                onChange={setDatabaseResources}
              />
            )}
            {radioValue === "EXPRESSION" && (
              <ExprEditor
                expr={expr}
                factorList={factorList}
                optionConfigMap={factorOptionConfigMap}
                factorOperatorOverrideMap={factorOperatorOverrideMap}
                onUpdate={setExpr}
              />
            )}
          </div>

          {/* Reason */}
          <FormField className="w-full" title={t("common.reason")}>
            <Input
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder={t("common.description")}
            />
          </FormField>

          {/* Expiration */}
          <FormField className="w-full" title={t("common.expiration")}>
            <FormControlGroup>
              <ExpirationPicker
                value={expirationTimestamp}
                onChange={setExpirationTimestamp}
                minDate={minDatetime}
              />
              {!expirationTimestamp && (
                <span className="textinfolabel">
                  {t("settings.sensitive-data.never-expires")}
                </span>
              )}
            </FormControlGroup>
          </FormField>

          {/* Members */}
          <FormField
            className="w-full"
            title={
              <span className="flex items-center gap-x-1">
                {t("settings.members.select-account", { count: 2 })}
                <span className="text-error">*</span>
                <Tooltip content={t("settings.members.select-account-hint")}>
                  <CircleHelp className="size-4 textinfolabel" />
                </Tooltip>
              </span>
            }
          >
            <AccountMultiSelect
              value={memberList}
              onChange={setMemberList}
              accountParents={accountParents}
            />
          </FormField>
        </div>
      </div>

      {/* Footer */}
      <StickyActionFooter
        left={
          <Button appearance="outline" onClick={onDismiss}>
            {t("common.cancel")}
          </Button>
        }
        right={
          <Button
            disabled={submitDisabled || processing || draftPreparing}
            onClick={onSubmit}
          >
            {t("common.confirm")}
          </Button>
        }
      />

      {/* Feature paywall — shared FeatureModal so the dialog content is
          driven by the subscription dynamic feature copy + plan info. */}
      <FeatureModal
        open={showFeatureModal}
        feature={PlanFeature.FEATURE_DATA_MASKING}
        onOpenChange={setShowFeatureModal}
      />
    </div>
  );
}
