import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import {
  ArrowDown,
  ArrowUp,
  ListOrdered,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react";
import {
  type Ref,
  useCallback,
  useEffect,
  useId,
  useImperativeHandle,
  useMemo,
  useRef,
  useState,
} from "react";
import { useTranslation } from "react-i18next";
import { v4 as uuidv4 } from "uuid";
import { settingServiceClientConnect } from "@/api";
import { router } from "@/app/router";
import { ExprEditor, type OptionConfig } from "@/components/ExprEditor";
import { FeatureAttention } from "@/components/FeatureAttention";
import { LearnMoreLink } from "@/components/LearnMoreLink";
import {
  SemanticTypeOption,
  SemanticTypeSelect,
} from "@/components/SemanticTypeSelect";
import { Alert } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { FormError, FormField, FormFieldGroup } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover";
import { Separator } from "@/components/ui/separator";
import { StickyActionFooter } from "@/components/ui/sticky-action-footer";
import { Tooltip } from "@/components/ui/tooltip";
import {
  WorkspacePageInfo,
  WorkspacePageLayout,
  WorkspacePageToolbar,
} from "@/components/WorkspacePageLayout";
import { useSemanticTypes } from "@/hooks/useSemanticTypes";
import { useUnsavedChangesGuard } from "@/hooks/useUnsavedChangesGuard";
import {
  factorOperatorOverrideMap,
  getClassificationLevelOptions,
} from "@/lib/sensitive-data/components-utils";
import { cn } from "@/lib/utils";
import { AgentHelpButton } from "@/modules/agent/components/AgentHelpButton";
import { openAgentPageHelp } from "@/modules/agent/logic/contextualHelp";
import { openAgentWithPrompt } from "@/modules/agent/logic/instanceConnectionHelp";
import {
  type MaskingRuleDraft,
  parseMaskingRuleDraftCondition,
  validateMaskingRuleDraft,
} from "@/modules/agent/logic/maskingRuleDraft";
import {
  type MaskingRuleEditorSnapshot,
  registerMaskingRuleDraftTarget,
} from "@/modules/agent/logic/maskingRuleDraftTarget";
import { type AgentState, useAgentStore } from "@/modules/agent/store/agent";
import type { ConditionGroupExpr, Factor, SimpleExpr } from "@/modules/cel";
import {
  buildCELExpr,
  ExprType,
  emptySimpleExpr,
  resolveCELExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/modules/cel";
import { pushNotification } from "@/stores";
import { useAppStore } from "@/stores/app";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import type {
  MaskingRulePolicy_MaskingRule,
  Policy,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  MaskingRulePolicy_MaskingRuleSchema,
  MaskingRulePolicySchema,
  PolicyResourceType,
  PolicyType,
} from "@/types/proto-es/v1/org_policy_service_pb";
import {
  GetSettingRequestSchema,
  Setting_SettingName,
} from "@/types/proto-es/v1/setting_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import {
  arraySwap,
  batchConvertCELStringToParsedExpr,
  batchConvertParsedExprToCELString,
  getDatabaseIdOptionConfig,
  getInstanceIdOptionConfig,
  getProjectIdOptionConfig,
  hasWorkspacePermissionV2,
} from "@/utils";
import {
  CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
  CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
  CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
  CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
  CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
  CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
  CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
} from "@/utils/cel-attributes";

type MaskingRuleMode = "NORMAL" | "EDIT" | "CREATE";
interface MaskingRuleItem {
  mode: MaskingRuleMode;
  rule: MaskingRulePolicy_MaskingRule;
  initialDraft?: {
    draft: MaskingRuleDraft;
    expr: ConditionGroupExpr;
    chatId: string;
  };
}

interface MaskingRuleEditorHandle {
  getLocalState: () => {
    title: string;
    semanticType: string;
    expr: ConditionGroupExpr;
  };
  openHelp: () => void;
  getSnapshot: () => Promise<MaskingRuleEditorSnapshot | undefined>;
  apply: (
    draft: MaskingRuleDraft,
    expr: ConditionGroupExpr,
    expected: MaskingRuleEditorSnapshot,
    chatId: string
  ) => Promise<void>;
}

function reviewChatUnavailable(state: AgentState, chatId: string): boolean {
  const chat = state.chats.find((item) => item.id === chatId);
  return (
    !chat ||
    chat.archived ||
    chat.status === "running" ||
    chat.status === "awaiting_user" ||
    !!state.pendingAskByChatId[chatId] ||
    !!state.queuedUserMessagesByChatId[chatId] ||
    !!state.queuedUserMessageInFlightByChatId[chatId] ||
    !!state.abortControllersByChatId[chatId]
  );
}

function singleRawCondition(expr: SimpleExpr): string | undefined {
  if (expr.type === ExprType.RawString) return expr.content;
  if (expr.type === ExprType.ConditionGroup && expr.args.length === 1)
    return singleRawCondition(expr.args[0]);
  return undefined;
}

async function parseRuleCondition(
  expression: string
): Promise<ConditionGroupExpr> {
  let resolved: SimpleExpr = expression
    ? { type: ExprType.RawString, content: expression }
    : emptySimpleExpr();
  try {
    if (expression) {
      const [parsed] = await batchConvertCELStringToParsedExpr([expression]);
      if (parsed?.exprKind.case) resolved = resolveCELExpr(parsed);
    }
  } catch {
    // Retain the original CEL if the selector cannot be loaded.
  }
  return wrapAsGroup(resolved);
}

function MaskingRuleCondition({
  expression,
  factorList,
  optionConfigMap,
}: {
  expression: string;
  factorList: Factor[];
  optionConfigMap: Map<Factor, OptionConfig>;
}) {
  const [expr, setExpr] = useState(() =>
    wrapAsGroup({ type: ExprType.RawString, content: expression })
  );
  useEffect(() => {
    let cancelled = false;
    void parseRuleCondition(expression).then((parsed) => {
      if (!cancelled) setExpr(parsed);
    });
    return () => {
      cancelled = true;
    };
  }, [expression]);

  return (
    <ExprEditor
      bordered={false}
      readonly
      expr={expr}
      factorList={factorList}
      optionConfigMap={optionConfigMap}
      factorOperatorOverrideMap={factorOperatorOverrideMap}
      onUpdate={() => {}}
    />
  );
}

function MaskingRuleEditor({
  index,
  disabled,
  mode,
  factorList,
  optionConfigMap,
  maskingRule,
  initialDraft,
  editorRef,
  onCancel,
  onDelete,
  onConfirm,
  onDirtyChange,
}: {
  index: number;
  disabled: boolean;
  mode: "EDIT" | "CREATE";
  factorList: Factor[];
  optionConfigMap: Map<Factor, OptionConfig>;
  maskingRule: MaskingRulePolicy_MaskingRule;
  initialDraft?: MaskingRuleItem["initialDraft"];
  editorRef: Ref<MaskingRuleEditorHandle>;
  onCancel: () => void;
  onDelete: () => Promise<boolean>;
  onConfirm: (rule: MaskingRulePolicy_MaskingRule) => Promise<boolean>;
  onDirtyChange: (dirty: boolean) => void;
}) {
  const { t } = useTranslation();
  const { semanticTypes } = useSemanticTypes();
  const [title, setTitle] = useState(
    initialDraft?.draft.title ?? maskingRule.condition?.title ?? ""
  );
  const [semanticType, setSemanticType] = useState(
    initialDraft?.draft.semanticType ?? maskingRule.semanticType ?? ""
  );
  const [expr, setExpr] = useState<ConditionGroupExpr>(
    initialDraft?.expr ?? wrapAsGroup(emptySimpleExpr())
  );
  const [savedExpr, setSavedExpr] = useState(() =>
    wrapAsGroup(emptySimpleExpr())
  );
  const [parsing, setParsing] = useState(!initialDraft);
  const [chatId, setChatId] = useState(initialDraft?.chatId);
  const [helpOwner] = useState(() => ({
    userName: useAppStore.getState().currentUser?.name,
    workspaceName: useAppStore.getState().workspaceResourceName(),
  }));
  const helpActive = useRef(true);
  const revokeHelp = useRef<(() => void) | undefined>(undefined);
  useEffect(() => {
    helpActive.current = true;
    return () => {
      helpActive.current = false;
      revokeHelp.current?.();
    };
  }, []);
  const [sessionId] = useState(() => uuidv4());
  const [preparing, setPreparing] = useState(false);
  const [conversionError, setConversionError] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const conditionErrorId = useId();
  const semanticErrorId = useId();
  const locked = disabled || parsing || preparing;

  useEffect(() => {
    if (initialDraft) return;
    let cancelled = false;
    const load = async () => {
      const group = await parseRuleCondition(
        maskingRule.condition?.expression ?? ""
      );
      if (cancelled) return;
      setExpr(group);
      setSavedExpr(group);
      setParsing(false);
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, [maskingRule, initialDraft]);

  const dirty =
    title !== (maskingRule.condition?.title ?? "") ||
    semanticType !== maskingRule.semanticType ||
    !isEqual(expr, savedExpr);
  useEffect(() => {
    onDirtyChange(dirty);
  }, [dirty, onDirtyChange]);

  const invalidCondition = !validateSimpleExpr(expr);
  const invalidSemanticType = !semanticTypes.some(
    (type) => type.id === semanticType
  );
  const invalidFields = invalidSemanticType || invalidCondition;
  const fieldSignature = JSON.stringify([title, semanticType, expr]);
  const revision = useRef({ signature: fieldSignature, value: 0 });
  if (revision.current.signature !== fieldSignature) {
    revision.current = {
      signature: fieldSignature,
      value: revision.current.value + 1,
    };
  }
  const currentFields = useRef({
    title,
    semanticType,
    expr,
    savedExpr,
    chatId,
    locked,
    revision: String(revision.current.value),
  });
  currentFields.current = {
    title,
    semanticType,
    expr,
    savedExpr,
    chatId,
    locked,
    revision: String(revision.current.value),
  };
  useImperativeHandle(
    editorRef,
    () => ({
      getLocalState: () => ({
        title: currentFields.current.title,
        semanticType: currentFields.current.semanticType,
        expr: structuredClone(currentFields.current.expr),
      }),
      openHelp: () => {
        const isAvailable = () => {
          const app = useAppStore.getState();
          return (
            helpActive.current &&
            app.currentUser?.name === helpOwner.userName &&
            app.workspaceResourceName() === helpOwner.workspaceName
          );
        };
        const canStart = () => {
          const fields = currentFields.current;
          return (
            !fields.locked &&
            (!fields.chatId ||
              !reviewChatUnavailable(useAgentStore.getState(), fields.chatId))
          );
        };
        revokeHelp.current?.();
        revokeHelp.current = openAgentPageHelp({
          page: { kind: "global-masking" },
          maskingRuleEditor: true,
          isAvailable,
          canStart,
          onStart: (prompt) => {
            if (!isAvailable() || !canStart()) return false;
            const text = `${prompt}\n${t("agent.masking-rule-draft.help-prompt")}`;
            const owner = currentFields.current.chatId;
            if (!owner) {
              openAgentWithPrompt(text, undefined, (id) => {
                currentFields.current.chatId = id;
                setChatId(id);
              });
              return true;
            }
            const store = useAgentStore.getState();
            store.setCurrentChat(owner);
            if (useAgentStore.getState().currentChatId !== owner) return false;
            store.queueUserMessage({
              chatId: owner,
              text,
              page: {
                path: router.currentRoute.value.fullPath,
                title: document.title,
              },
              runMode: "read-only",
            });
            store.open();
            return true;
          },
        });
      },
      getSnapshot: async () => {
        const fields = currentFields.current;
        if (fields.locked) return undefined;
        const rawCondition = singleRawCondition(fields.expr);
        let condition = rawCondition ?? "";
        let incomplete = !validateSimpleExpr(fields.expr);
        try {
          if (!incomplete && !isEqual(fields.expr, fields.savedExpr)) {
            const parsed = await buildCELExpr(fields.expr);
            if (parsed) {
              const [converted] = await batchConvertParsedExprToCELString([
                parsed,
              ]);
              condition = rawCondition ?? converted ?? "";
              incomplete = !converted;
            } else incomplete = true;
          } else if (!incomplete) {
            condition = rawCondition ?? maskingRule.condition?.expression ?? "";
          }
        } catch {
          return undefined;
        }
        incomplete ||= !condition;
        if (
          fields.revision !== currentFields.current.revision ||
          currentFields.current.locked ||
          fields.title.length > 200 ||
          condition.length > 20000 ||
          fields.semanticType.length > 200
        )
          return undefined;
        return {
          mode: mode === "EDIT" ? "edit" : "create",
          sessionId,
          revision: fields.revision,
          title: fields.title,
          condition,
          semanticType: fields.semanticType,
          chatId: fields.chatId,
          ...(incomplete ? { conditionStatus: "incomplete" as const } : {}),
        };
      },
      apply: async (draft, nextExpr, expected, originatingChatId) => {
        const fields = currentFields.current;
        if (
          expected.sessionId !== sessionId ||
          expected.revision !== fields.revision ||
          expected.chatId !== fields.chatId ||
          (fields.chatId && fields.chatId !== originatingChatId)
        )
          throw new Error(
            "The masking rule draft has changed. Prepare it again."
          );
        setTitle(draft.title);
        setSemanticType(draft.semanticType);
        setExpr(nextExpr);
        setChatId(originatingChatId);
        setConversionError(false);
        const signature = JSON.stringify([
          draft.title,
          draft.semanticType,
          nextExpr,
        ]);
        if (revision.current.signature !== signature) {
          revision.current = {
            signature,
            value: revision.current.value + 1,
          };
        }
        currentFields.current = {
          ...fields,
          title: draft.title,
          semanticType: draft.semanticType,
          expr: nextExpr,
          chatId: originatingChatId,
          revision: String(revision.current.value),
        };
      },
    }),
    [helpOwner, maskingRule, mode, sessionId, t]
  );
  const showConditionError = !parsing && (invalidCondition || conversionError);
  const showSemanticError = !parsing && invalidSemanticType;
  const handleConfirm = async () => {
    if (locked || !dirty || invalidFields) return;
    setPreparing(true);
    setConversionError(false);
    try {
      let expression = maskingRule.condition?.expression ?? "";
      // Renaming or changing semantic type must not rewrite an untouched expression.
      if (!isEqual(expr, savedExpr)) {
        const celExpr = await buildCELExpr(expr);
        if (!celExpr) {
          setConversionError(true);
          return;
        }
        const [converted] = await batchConvertParsedExprToCELString([celExpr]);
        if (!converted) {
          setConversionError(true);
          return;
        }
        expression = converted;
      }
      await onConfirm({
        ...maskingRule,
        semanticType,
        condition: maskingRule.condition
          ? { ...maskingRule.condition, expression, title }
          : create(ExprSchema, { expression, title }),
      });
    } catch {
      setConversionError(true);
    } finally {
      setPreparing(false);
    }
  };

  const cancelButton = (
    <Button appearance="outline" disabled={locked} onClick={onCancel}>
      {t("common.cancel")}
    </Button>
  );
  const saveButton = (
    <Button
      disabled={locked || !dirty || invalidFields}
      onClick={handleConfirm}
    >
      {mode === "CREATE"
        ? t("settings.sensitive-data.global-rules.create-rule")
        : t("settings.sensitive-data.global-rules.save-rule")}
    </Button>
  );
  const deleteButton = mode === "EDIT" && (
    <Popover open={showDeleteConfirm} onOpenChange={setShowDeleteConfirm}>
      <PopoverTrigger
        disabled={locked}
        render={<Button appearance="secondary" variant="destructive" />}
      >
        <Trash2 className="size-4" />
        {t("settings.sensitive-data.global-rules.delete-rule")}
      </PopoverTrigger>
      <PopoverContent aria-label={t("common.delete")}>
        <p className="mb-3 text-sm">
          {t("settings.sensitive-data.global-rules.delete-rule-tip")}
        </p>
        <div className="flex justify-end gap-2">
          <Button
            appearance="outline"
            size="sm"
            disabled={locked}
            onClick={() => setShowDeleteConfirm(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button
            variant="destructive"
            size="sm"
            disabled={locked}
            onClick={async () => {
              if (await onDelete()) setShowDeleteConfirm(false);
            }}
          >
            {t("common.delete")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
  const saveActions = (
    <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
      {cancelButton}
      {saveButton}
    </div>
  );

  return (
    <div className="w-full min-w-0">
      <FormFieldGroup>
        <FormField title={t("settings.sensitive-data.global-rules.name")}>
          <Input
            autoFocus={!!initialDraft}
            aria-label={t("settings.sensitive-data.global-rules.name")}
            value={title}
            disabled={locked}
            placeholder={`${t("settings.sensitive-data.global-rules.condition-order")} ${index}`}
            onChange={(event) => setTitle(event.target.value)}
          />
        </FormField>
        <FormField
          title={t("settings.sensitive-data.global-rules.condition")}
          role="group"
          aria-label={t("settings.sensitive-data.global-rules.condition")}
          aria-invalid={showConditionError || undefined}
          aria-describedby={showConditionError ? conditionErrorId : undefined}
        >
          <ExprEditor
            expr={expr}
            readonly={locked}
            factorList={factorList}
            optionConfigMap={optionConfigMap}
            factorOperatorOverrideMap={factorOperatorOverrideMap}
            onUpdate={setExpr}
          />
          {showConditionError && (
            <FormError id={conditionErrorId}>
              {t(
                "settings.sensitive-data.global-rules.error.invalid-condition"
              )}
            </FormError>
          )}
        </FormField>
        <FormField
          title={t("settings.sensitive-data.global-rules.semantic-type")}
        >
          <SemanticTypeSelect
            aria-label={t("settings.sensitive-data.global-rules.semantic-type")}
            aria-invalid={showSemanticError || undefined}
            aria-describedby={showSemanticError ? semanticErrorId : undefined}
            id={`masking-semantic-type-${maskingRule.id}`}
            value={semanticType}
            disabled={locked}
            className="w-full max-w-sm"
            onValueChange={setSemanticType}
          />
          {showSemanticError && (
            <FormError id={semanticErrorId}>
              {t(
                "settings.sensitive-data.global-rules.error.missing-semantic-type"
              )}
            </FormError>
          )}
        </FormField>
      </FormFieldGroup>
      {dirty ? (
        <StickyActionFooter
          className="mt-6"
          contentClassName="flex-wrap"
          contentPadding={false}
          rightClassName="ml-auto"
          left={deleteButton}
          right={saveActions}
        />
      ) : (
        <div className="mt-6 flex flex-wrap items-center justify-between gap-2">
          {deleteButton}
          {saveActions}
        </div>
      )}
    </div>
  );
}

export function GlobalMaskingPage() {
  const { t } = useTranslation();
  const hasSensitiveDataFeature = useAppStore((s) =>
    s.hasInstanceFeature(PlanFeature.FEATURE_DATA_MASKING)
  );
  const hasPermission = hasWorkspacePermissionV2(
    "bb.policies.updateMaskingRulePolicy"
  );
  const canEdit = hasPermission && hasSensitiveDataFeature;
  const { semanticTypes } = useSemanticTypes();
  const [items, setItems] = useState<MaskingRuleItem[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [processing, setProcessing] = useState(false);
  const [reorderRules, setReorderRules] = useState(false);
  const [draftDirty, setDraftDirty] = useState(false);
  const [mutationError, setMutationError] = useState(false);
  const [draftPreparing, setDraftPreparing] = useState(false);
  type QueuedDraft = {
    id: string;
    draft: MaskingRuleDraft;
    expr: ConditionGroupExpr;
    status: "pending" | "editing" | "created";
  };
  const [ruleDrafts, setRuleDrafts] = useState<QueuedDraft[]>([]);
  const queue = useRef<
    | {
        drafts: QueuedDraft[];
        chatId: string;
        workspaceName: string;
        userName: string;
        baseline: { rules: MaskingRulePolicy_MaskingRule[]; semantic: unknown };
      }
    | undefined
  >(undefined);
  const [queueNeedsRecheck, setQueueNeedsRecheck] = useState(false);
  const updateQueue = (next: QueuedDraft[]) => {
    if (queue.current) queue.current.drafts = next;
    setRuleDrafts(next);
  };
  const draftOperation = useRef(false);
  const pageActive = useRef(true);
  useEffect(() => {
    pageActive.current = true;
    return () => {
      pageActive.current = false;
    };
  }, []);
  const editorRef = useRef<MaskingRuleEditorHandle>(null);
  const workspaceName = useAppStore((s) => s.workspaceResourceName());
  const userName = useAppStore((s) => s.currentUser?.name);
  const reorderSnapshot = useRef<MaskingRuleItem[]>([]);
  const pageState = useRef({
    items,
    loaded,
    canEdit,
    processing,
    reorderRules,
  });
  pageState.current = { items, loaded, canEdit, processing, reorderRules };
  const updateItems = (next: MaskingRuleItem[]) => {
    pageState.current.items = next;
    setItems(next);
  };
  const activeEditor = items.some((item) => item.mode !== "NORMAL");
  const hasPendingQueue = ruleDrafts.some(
    (draft) => draft.status !== "created"
  );
  const reorderDirty =
    reorderRules &&
    !isEqual(
      items.map((item) => item.rule.id),
      reorderSnapshot.current.map((item) => item.rule.id)
    );
  useUnsavedChangesGuard(
    (activeEditor && draftDirty) ||
      reorderDirty ||
      ruleDrafts.some((draft) => draft.status !== "created")
  );

  const assertQueueOwner = () => {
    const owner = queue.current;
    const app = useAppStore.getState();
    const agent = useAgentStore.getState();
    const chat = owner && agent.getChat(owner.chatId);
    if (
      !owner ||
      !pageActive.current ||
      app.workspaceResourceName() !== owner.workspaceName ||
      app.currentUser?.name !== owner.userName ||
      agent.currentChatId !== owner.chatId ||
      !chat ||
      chat.archived ||
      chat.workspaceName !== owner.workspaceName ||
      !hasWorkspacePermissionV2("bb.policies.updateMaskingRulePolicy") ||
      !app.hasInstanceFeature(PlanFeature.FEATURE_DATA_MASKING)
    )
      throw new Error("The masking rule draft queue is no longer current.");
    return owner;
  };
  const readQueueBaseline = async () => {
    const app = useAppStore.getState();
    const policy = await app.getOrFetchPolicyByParentAndType({
      parentPath: workspaceName,
      policyType: PolicyType.MASKING_RULE,
      refresh: true,
    });
    const setting = await settingServiceClientConnect.getSetting(
      create(GetSettingRequestSchema, { name: "settings/SEMANTIC_TYPES" })
    );
    if (
      app.workspaceResourceName() !== workspaceName ||
      useAppStore.getState().currentUser?.name !== userName
    )
      throw new Error("The masking rule draft context is no longer current.");
    useAppStore.getState().setSettingByName(setting);
    return {
      rules: structuredClone(
        policy?.policy.case === "maskingRulePolicy"
          ? policy.policy.value.rules
          : []
      ),
      semantic: structuredClone(setting.value),
    };
  };
  const captureQueueEditor = () => {
    const fields = editorRef.current?.getLocalState();
    const owner = queue.current;
    if (!fields || !owner) return;
    updateQueue(
      owner.drafts.map((entry) =>
        entry.status === "editing"
          ? {
              ...entry,
              draft: {
                ...entry.draft,
                title: fields.title,
                semanticType: fields.semanticType,
              },
              expr: fields.expr,
            }
          : entry
      )
    );
  };
  const checkQueue = async (recheck = false) => {
    const owner = assertQueueOwner();
    if (!(await useAppStore.getState().refreshSubscription()))
      throw new Error("The masking feature could not be verified.");
    const baseline = await readQueueBaseline();
    assertQueueOwner();
    if (!recheck && (queueNeedsRecheck || !isEqual(baseline, owner.baseline))) {
      setQueueNeedsRecheck(true);
      throw new Error("Recheck the masking rule drafts before continuing.");
    }
    for (const entry of owner.drafts.filter(
      (draft) => draft.status !== "created"
    )) {
      await validateMaskingRuleDraft(
        entry.draft,
        {
          workspaceName: owner.workspaceName,
          userName: owner.userName,
          chatId: owner.chatId,
        },
        { purpose: "review" }
      );
      assertQueueOwner();
    }
    if (recheck) {
      owner.baseline = baseline;
      const active = pageState.current.items.filter(
        (item) => item.mode !== "NORMAL"
      );
      updateItems([
        ...baseline.rules.map((rule) => ({ mode: "NORMAL" as const, rule })),
        ...active,
      ]);
      setQueueNeedsRecheck(false);
    }
    return owner;
  };

  const draftCallbacks = useRef({
    getSnapshot: async (): Promise<MaskingRuleEditorSnapshot | undefined> =>
      undefined,
    open: async (
      _draft: MaskingRuleDraft,
      _chatId: string,
      _active: () => boolean
    ) => {},
    openMany: async (
      _drafts: MaskingRuleDraft[],
      _chatId: string,
      _active: () => boolean
    ) => {},
    apply: async (
      _draft: MaskingRuleDraft,
      _expected: MaskingRuleEditorSnapshot,
      _chatId: string,
      _active: () => boolean
    ) => {},
  });
  const assertDraftAvailable = (
    active: () => boolean,
    expectsEditor: boolean,
    chatId: string,
    allowExisting = false
  ) => {
    const current = pageState.current;
    const app = useAppStore.getState();
    const agent = useAgentStore.getState();
    const chat = agent.getChat(chatId);
    if (
      !active() ||
      !current.loaded ||
      !current.canEdit ||
      current.processing ||
      current.reorderRules ||
      current.items.some((item) => item.mode !== "NORMAL") !== expectsEditor ||
      (expectsEditor &&
        !allowExisting &&
        !current.items.some((item) => item.mode === "CREATE")) ||
      agent.currentChatId !== chatId ||
      !chat ||
      chat.archived ||
      chat.workspaceName !== workspaceName ||
      app.workspaceResourceName() !== workspaceName ||
      app.currentUser?.name !== userName
    )
      throw new Error(
        "The masking rule editor is unavailable or already in use."
      );
  };
  draftCallbacks.current = {
    getSnapshot: async () => {
      if (
        draftOperation.current ||
        pageState.current.processing ||
        pageState.current.reorderRules ||
        !pageState.current.items.some((item) => item.mode !== "NORMAL")
      )
        return undefined;
      const chatId = useAgentStore.getState().currentChatId;
      if (!chatId) return undefined;
      try {
        assertDraftAvailable(() => true, true, chatId, true);
        const editor = editorRef.current;
        const snapshot = await editor?.getSnapshot();
        assertDraftAvailable(() => true, true, chatId, true);
        if (editor !== editorRef.current) return undefined;
        return snapshot;
      } catch {
        return undefined;
      }
    },
    open: async (draft, chatId, active) => {
      if (queue.current?.drafts.some((entry) => entry.status !== "created"))
        throw new Error("Finish the current masking rule draft queue first.");
      assertDraftAvailable(active, false, chatId);
      if (draftOperation.current)
        throw new Error("The masking rule editor is busy.");
      draftOperation.current = true;
      setDraftPreparing(true);
      try {
        const expr = await parseMaskingRuleDraftCondition(draft.condition);
        assertDraftAvailable(active, false, chatId);
        setMutationError(false);
        setDraftDirty(true);
        updateItems([
          ...pageState.current.items,
          {
            mode: "CREATE",
            rule: create(MaskingRulePolicy_MaskingRuleSchema, { id: uuidv4() }),
            initialDraft: { draft, expr, chatId },
          },
        ]);
      } finally {
        draftOperation.current = false;
        setDraftPreparing(false);
      }
    },
    openMany: async (drafts, chatId, active) => {
      assertDraftAvailable(active, false, chatId);
      if (
        draftOperation.current ||
        queue.current?.drafts.some((entry) => entry.status !== "created")
      )
        throw new Error("The masking rule editor is busy.");
      draftOperation.current = true;
      setDraftPreparing(true);
      try {
        if (!(await useAppStore.getState().refreshSubscription()))
          throw new Error("The masking feature could not be verified.");
        const parsed: QueuedDraft[] = [];
        for (const [index, draft] of drafts.entries()) {
          await validateMaskingRuleDraft(draft, {
            workspaceName,
            userName: userName!,
            chatId,
          });
          parsed.push({
            id: `${uuidv4()}-${index}`,
            draft: structuredClone(draft),
            expr: await parseMaskingRuleDraftCondition(draft.condition),
            status: index === 0 ? "editing" : "pending",
          });
        }
        const baseline = await readQueueBaseline();
        assertDraftAvailable(active, false, chatId);
        queue.current = {
          drafts: parsed,
          chatId,
          workspaceName,
          userName: userName!,
          baseline,
        };
        updateQueue(parsed);
        setQueueNeedsRecheck(false);
        setMutationError(false);
        setDraftDirty(true);
        const first = parsed[0];
        updateItems([
          ...baseline.rules.map((rule) => ({ mode: "NORMAL" as const, rule })),
          {
            mode: "CREATE",
            rule: create(MaskingRulePolicy_MaskingRuleSchema, { id: first.id }),
            initialDraft: { draft: first.draft, expr: first.expr, chatId },
          },
        ]);
      } finally {
        draftOperation.current = false;
        setDraftPreparing(false);
      }
    },
    apply: async (draft, expected, chatId, active) => {
      assertDraftAvailable(active, true, chatId);
      if (draftOperation.current)
        throw new Error("The masking rule editor is busy.");
      const editor = editorRef.current;
      if (!editor) throw new Error("The masking rule editor is not ready.");
      draftOperation.current = true;
      setDraftPreparing(true);
      try {
        const expr = await parseMaskingRuleDraftCondition(draft.condition);
        assertDraftAvailable(active, true, chatId);
        if (editor !== editorRef.current)
          throw new Error("The masking rule editor has changed.");
        await editor.apply(draft, expr, expected, chatId);
        const activeRuleId = pageState.current.items.find(
          (item) => item.mode === "CREATE"
        )?.rule.id;
        if (queue.current?.drafts.some((entry) => entry.id === activeRuleId)) {
          updateQueue(
            queue.current.drafts.map((entry) =>
              entry.id === activeRuleId
                ? { ...entry, draft: structuredClone(draft), expr }
                : entry
            )
          );
        }
        setDraftDirty(true);
        setMutationError(false);
      } finally {
        draftOperation.current = false;
        setDraftPreparing(false);
      }
    },
  };
  useEffect(() => {
    if (!loaded || !canEdit || !workspaceName || !userName) return;
    let active = true;
    const unregister = registerMaskingRuleDraftTarget(
      { workspaceName, userName },
      {
        getSnapshot: () => draftCallbacks.current.getSnapshot(),
        open: (draft, chatId) =>
          draftCallbacks.current.open(draft, chatId, () => active),
        openMany: (drafts, chatId) =>
          draftCallbacks.current.openMany(drafts, chatId, () => active),
        apply: (draft, expected, chatId) =>
          draftCallbacks.current.apply(draft, expected, chatId, () => active),
      }
    );
    return () => {
      active = false;
      unregister();
    };
  }, [loaded, canEdit, workspaceName, userName]);

  const factorList = useMemo(
    (): Factor[] => [
      CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID,
      CEL_ATTRIBUTE_RESOURCE_PROJECT_ID,
      CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID,
      CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME,
      CEL_ATTRIBUTE_RESOURCE_TABLE_NAME,
      CEL_ATTRIBUTE_RESOURCE_COLUMN_NAME,
      CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL,
    ],
    []
  );
  const environmentList = useAppStore((s) => s.environmentList);
  const environmentOptions = useMemo(
    () =>
      environmentList.map((env) => ({
        label: `${env.title} (${env.id})`,
        value: env.id,
      })),
    [environmentList]
  );
  const classificationSettingsByName = useAppStore((s) => s.settingsByName);
  const classificationOptions = useMemo(
    () => getClassificationLevelOptions(),
    [classificationSettingsByName]
  );
  const factorOptionsMap = useMemo((): Map<Factor, OptionConfig> => {
    const workspaceName = useAppStore.getState().workspaceResourceName();
    return factorList.reduce((map, factor) => {
      switch (factor) {
        case CEL_ATTRIBUTE_RESOURCE_ENVIRONMENT_ID:
          map.set(factor, { options: environmentOptions });
          break;
        case CEL_ATTRIBUTE_RESOURCE_INSTANCE_ID:
          map.set(factor, getInstanceIdOptionConfig());
          break;
        case CEL_ATTRIBUTE_RESOURCE_PROJECT_ID:
          map.set(factor, getProjectIdOptionConfig());
          break;
        case CEL_ATTRIBUTE_RESOURCE_CLASSIFICATION_LEVEL:
          map.set(factor, { options: classificationOptions });
          break;
        case CEL_ATTRIBUTE_RESOURCE_DATABASE_NAME:
          map.set(factor, getDatabaseIdOptionConfig(workspaceName));
          break;
        default:
          map.set(factor, { options: [] });
      }
      return map;
    }, new Map<Factor, OptionConfig>());
  }, [factorList, environmentOptions, classificationOptions]);

  useEffect(() => {
    let cancelled = false;
    const load = async () => {
      const policy = await useAppStore
        .getState()
        .getOrFetchPolicyByParentAndType({
          parentPath: useAppStore.getState().workspaceResourceName(),
          policyType: PolicyType.MASKING_RULE,
        });
      if (cancelled) return;
      const rules =
        policy?.policy?.case === "maskingRulePolicy"
          ? policy.policy.value.rules
          : [];
      updateItems(rules.map((rule) => ({ mode: "NORMAL", rule })));
      setLoaded(true);
      await Promise.all([
        useAppStore
          .getState()
          .getOrFetchSettingByName(Setting_SettingName.SEMANTIC_TYPES, true),
        useAppStore
          .getState()
          .getOrFetchSettingByName(
            Setting_SettingName.DATA_CLASSIFICATION,
            true
          ),
      ]);
    };
    void load();
    return () => {
      cancelled = true;
    };
  }, []);

  const upsertPolicy = useCallback(async (currentItems: MaskingRuleItem[]) => {
    const patch: Partial<Policy> = {
      type: PolicyType.MASKING_RULE,
      resourceType: PolicyResourceType.WORKSPACE,
      policy: {
        case: "maskingRulePolicy",
        value: create(MaskingRulePolicySchema, {
          rules: currentItems
            .filter((item) => item.mode === "NORMAL")
            .map((item) => item.rule),
        }),
      },
    };
    await useAppStore.getState().upsertPolicy({
      parentPath: useAppStore.getState().workspaceResourceName(),
      policy: patch,
    });
  }, []);

  const persist = async (
    nextItems: MaskingRuleItem[],
    notification: string
  ) => {
    if (
      pageState.current.processing ||
      !pageState.current.canEdit ||
      draftOperation.current
    )
      return false;
    pageState.current.processing = true;
    setProcessing(true);
    setMutationError(false);
    try {
      await upsertPolicy(nextItems);
      updateItems(nextItems);
      setDraftDirty(false);
      pushNotification({
        module: "bytebase",
        style: "SUCCESS",
        title: notification,
      });
      return true;
    } catch {
      setMutationError(true);
      return false;
    } finally {
      setProcessing(false);
      pageState.current.processing = false;
    }
  };
  const addNewRule = () => {
    const current = pageState.current;
    if (
      !current.canEdit ||
      !current.loaded ||
      current.items.some((item) => item.mode !== "NORMAL") ||
      current.reorderRules ||
      current.processing ||
      draftOperation.current ||
      hasPendingQueue
    )
      return;
    setMutationError(false);
    setDraftDirty(false);
    updateItems([
      ...pageState.current.items,
      {
        mode: "CREATE",
        rule: create(MaskingRulePolicy_MaskingRuleSchema, { id: uuidv4() }),
      },
    ]);
  };
  const onEdit = (index: number) => {
    const current = pageState.current;
    if (
      !current.canEdit ||
      current.items.some((item) => item.mode !== "NORMAL") ||
      current.reorderRules ||
      current.processing ||
      draftOperation.current ||
      hasPendingQueue
    )
      return;
    setMutationError(false);
    setDraftDirty(false);
    updateItems(
      pageState.current.items.map((item, i) =>
        i === index ? { ...item, mode: "EDIT" } : item
      )
    );
  };
  const onCancel = (index: number) => {
    if (pageState.current.processing || draftOperation.current) return;
    const cancelledId = pageState.current.items[index]?.rule.id;
    if (queue.current?.drafts.some((entry) => entry.id === cancelledId))
      updateQueue(
        queue.current.drafts.map((entry) =>
          entry.id === cancelledId ? { ...entry, status: "pending" } : entry
        )
      );
    updateItems(
      pageState.current.items.flatMap((item, i) =>
        i !== index
          ? [item]
          : item.mode === "CREATE"
            ? []
            : [{ ...item, mode: "NORMAL" as const }]
      )
    );
    setDraftDirty(false);
    setMutationError(false);
  };
  const onConfirm = async (rule: MaskingRulePolicy_MaskingRule) => {
    const index = items.findIndex((item) => item.rule.id === rule.id);
    if (index < 0) return false;
    const isCreate = items[index].mode === "CREATE";
    const queued = queue.current?.drafts.find((entry) => entry.id === rule.id);
    if (queued) {
      if (draftOperation.current || pageState.current.processing) return false;
      captureQueueEditor();
      draftOperation.current = true;
      try {
        await checkQueue();
        const owner = assertQueueOwner();
        const validated = await validateMaskingRuleDraft(
          {
            ...queued.draft,
            title: rule.condition?.title || queued.draft.title,
            semanticType: rule.semanticType,
            condition: rule.condition?.expression ?? "",
          },
          {
            workspaceName: owner.workspaceName,
            userName: owner.userName,
            chatId: owner.chatId,
          }
        );
        assertQueueOwner();
        if (!isEqual(await readQueueBaseline(), owner.baseline)) {
          setQueueNeedsRecheck(true);
          throw new Error("Recheck the masking rule drafts before continuing.");
        }
        assertQueueOwner();
        if (rule.condition)
          rule = {
            ...rule,
            condition: { ...rule.condition, expression: validated.condition },
          };
      } catch {
        setMutationError(true);
        return false;
      } finally {
        draftOperation.current = false;
      }
    }
    const next = items.map((item, i) =>
      i === index ? { mode: "NORMAL" as const, rule } : item
    );
    const saved = await persist(
      next,
      t(isCreate ? "common.created" : "common.updated")
    );
    if (saved && queued && queue.current) {
      queue.current.baseline.rules = structuredClone(
        next.filter((item) => item.mode === "NORMAL").map((item) => item.rule)
      );
      updateQueue(
        queue.current.drafts.map((entry) =>
          entry.id === rule.id ? { ...entry, status: "created" } : entry
        )
      );
    }
    return saved;
  };
  const onRuleDelete = (index: number) =>
    persist(
      items.filter((_, i) => i !== index),
      t("common.deleted")
    );
  const onReorder = (index: number, offset: number) => {
    if (
      processing ||
      draftOperation.current ||
      !reorderRules ||
      index + offset < 0 ||
      index + offset >= items.length
    )
      return;
    const next = [...pageState.current.items];
    arraySwap(next, index, index + offset);
    updateItems(next);
  };
  const startReorder = () => {
    const current = pageState.current;
    if (
      !current.canEdit ||
      !current.loaded ||
      current.processing ||
      current.items.some((item) => item.mode !== "NORMAL") ||
      current.items.length < 2 ||
      draftOperation.current ||
      hasPendingQueue
    )
      return;
    reorderSnapshot.current = current.items;
    setMutationError(false);
    setReorderRules(true);
    pageState.current.reorderRules = true;
  };
  const cancelReorder = () => {
    if (pageState.current.processing || draftOperation.current) return;
    updateItems(reorderSnapshot.current);
    setReorderRules(false);
    pageState.current.reorderRules = false;
    setMutationError(false);
  };
  const saveReorder = async () => {
    if (!reorderDirty) return;
    if (await persist(items, t("common.updated"))) setReorderRules(false);
  };

  const reviewQueuedDraft = async (id: string) => {
    if (draftOperation.current || pageState.current.processing) return;
    captureQueueEditor();
    draftOperation.current = true;
    setDraftPreparing(true);
    try {
      const owner = await checkQueue();
      const selected = owner.drafts.find(
        (entry) => entry.id === id && entry.status === "pending"
      );
      if (!selected) return;
      updateQueue(
        owner.drafts.map((entry) =>
          entry.id === id
            ? { ...entry, status: "editing" }
            : entry.status === "editing"
              ? { ...entry, status: "pending" }
              : entry
        )
      );
      updateItems([
        ...pageState.current.items.filter((item) => item.mode === "NORMAL"),
        {
          mode: "CREATE",
          rule: create(MaskingRulePolicy_MaskingRuleSchema, { id }),
          initialDraft: {
            draft: selected.draft,
            expr: selected.expr,
            chatId: owner.chatId,
          },
        },
      ]);
      setDraftDirty(true);
      setMutationError(false);
    } catch {
      setMutationError(true);
    } finally {
      draftOperation.current = false;
      setDraftPreparing(false);
    }
  };

  return (
    <WorkspacePageLayout>
      <FeatureAttention feature={PlanFeature.FEATURE_DATA_MASKING} />
      <WorkspacePageInfo
        description={
          <>
            {hasSensitiveDataFeature && (
              <>{t("custom-approval.rule.first-match-wins")} </>
            )}
            {t("settings.sensitive-data.global-rules.description")}{" "}
            <LearnMoreLink href="https://docs.bytebase.com/security/data-masking/overview/?source=console" />
          </>
        }
      />
      <div className="flex min-w-0 flex-col gap-4">
        <WorkspacePageToolbar className="flex-wrap gap-y-3">
          <span className="shrink-0 whitespace-nowrap text-sm text-control-light">
            {t("settings.sensitive-data.global-rules.rule-count", {
              count: items.length,
            })}
          </span>
          <div className="ml-auto flex min-w-0 max-w-full flex-wrap items-center justify-end gap-2">
            <AgentHelpButton
              onClick={() => {
                if (
                  pageState.current.items.some((item) => item.mode !== "NORMAL")
                ) {
                  editorRef.current?.openHelp();
                } else {
                  openAgentPageHelp({ page: { kind: "global-masking" } });
                }
              }}
            >
              {t("agent.masking.ask-agent-short")}
            </AgentHelpButton>
            {reorderRules ? (
              <>
                <Button
                  appearance="outline"
                  disabled={processing || draftPreparing}
                  onClick={cancelReorder}
                >
                  {t("common.cancel")}
                </Button>
                <Button
                  disabled={processing || draftPreparing || !reorderDirty}
                  onClick={saveReorder}
                >
                  {t("settings.sensitive-data.global-rules.save-order")}
                </Button>
              </>
            ) : (
              <>
                <Button
                  appearance="outline"
                  disabled={
                    !canEdit ||
                    !loaded ||
                    activeEditor ||
                    processing ||
                    draftPreparing ||
                    hasPendingQueue ||
                    items.length <= 1
                  }
                  onClick={startReorder}
                >
                  <ListOrdered className="size-4" />
                  {t("settings.sensitive-data.global-rules.re-order")}
                </Button>
                <Button
                  disabled={
                    !canEdit ||
                    !loaded ||
                    activeEditor ||
                    processing ||
                    draftPreparing ||
                    hasPendingQueue
                  }
                  onClick={addNewRule}
                >
                  <Plus className="size-4" />
                  {t("settings.sensitive-data.global-rules.create-rule")}
                </Button>
              </>
            )}
          </div>
        </WorkspacePageToolbar>
        {mutationError && (
          <Alert
            variant="error"
            description={t(
              "settings.sensitive-data.global-rules.error.save-failed"
            )}
          />
        )}
        {ruleDrafts.length > 0 && (
          <section
            aria-label={t("agent.masking-rule-draft.queue.title")}
            className="flex min-w-0 flex-col gap-2"
          >
            <h2 className="text-base font-semibold">
              {t("agent.masking-rule-draft.queue.title")}
            </h2>
            {queueNeedsRecheck && (
              <Alert
                variant="warning"
                description={t("agent.masking-rule-draft.queue.needs-recheck")}
              />
            )}
            {queueNeedsRecheck && (
              <Button
                appearance="ghost"
                size="sm"
                className="self-start"
                disabled={draftPreparing || processing || !canEdit}
                onClick={async () => {
                  if (draftOperation.current) return;
                  captureQueueEditor();
                  draftOperation.current = true;
                  setDraftPreparing(true);
                  try {
                    await checkQueue(true);
                    setMutationError(false);
                  } catch {
                    setMutationError(true);
                  } finally {
                    draftOperation.current = false;
                    setDraftPreparing(false);
                  }
                }}
              >
                {t("agent.masking-rule-draft.queue.recheck")}
              </Button>
            )}
            <ul className="flex min-w-0 flex-col gap-2">
              {ruleDrafts.map((entry) => (
                <li
                  key={entry.id}
                  className="flex min-w-0 flex-wrap items-center gap-2"
                >
                  <div className="flex min-w-0 flex-1 flex-col gap-1">
                    <span className="wrap-anywhere text-sm">
                      {entry.draft.title}
                    </span>
                    <dl className="flex min-w-0 flex-wrap items-baseline gap-x-2 text-xs">
                      <dt className="text-control-light">
                        {t("agent.masking-rule-draft.scope")}
                      </dt>
                      <dd className="min-w-0 wrap-anywhere">
                        {entry.draft.scope.kind === "workspace"
                          ? t("agent.masking-rule-draft.all-databases")
                          : entry.draft.scope.kind === "project"
                            ? entry.draft.scope.project
                            : entry.draft.scope.database}
                      </dd>
                    </dl>
                  </div>
                  <span className="text-xs text-control-light wrap-anywhere">
                    {semanticTypes.find(
                      (type) => type.id === entry.draft.semanticType
                    )?.title ?? entry.draft.semanticType}
                  </span>
                  <Badge
                    variant={
                      entry.status === "created"
                        ? "success"
                        : entry.status === "editing"
                          ? "secondary"
                          : "default"
                    }
                  >
                    {entry.status === "created"
                      ? t("agent.masking-rule-draft.queue.created")
                      : entry.status === "editing"
                        ? t("agent.masking-rule-draft.queue.editing")
                        : t("agent.masking-rule-draft.queue.pending")}
                  </Badge>
                  {entry.status === "pending" && (
                    <Button
                      appearance="ghost"
                      size="sm"
                      disabled={
                        draftPreparing ||
                        processing ||
                        queueNeedsRecheck ||
                        !canEdit
                      }
                      onClick={() => {
                        void reviewQueuedDraft(entry.id);
                      }}
                    >
                      {t("agent.masking-rule-draft.queue.review")}
                    </Button>
                  )}
                </li>
              ))}
            </ul>
            <Separator />
          </section>
        )}
        {!loaded ? (
          <p className="text-sm text-control-light">{t("common.loading")}</p>
        ) : items.length === 0 ? (
          <p className="py-8 text-sm text-control-light">
            {t("common.no-data")}
          </p>
        ) : (
          <ol className="m-0 flex min-w-0 list-decimal flex-col gap-4 pl-6 text-sm marker:text-control-light">
            {items.map((item, index) => {
              const semanticType = semanticTypes.find(
                (type) => type.id === item.rule.semanticType
              );
              const title =
                item.rule.condition?.title ||
                `${t("settings.sensitive-data.global-rules.condition-order")} ${index + 1}`;
              return (
                <li key={item.rule.id} className="min-w-0 pl-2">
                  {item.mode !== "NORMAL" ? (
                    <MaskingRuleEditor
                      key={item.rule.id}
                      index={index + 1}
                      mode={item.mode}
                      disabled={processing || draftPreparing || !canEdit}
                      maskingRule={item.rule}
                      initialDraft={item.initialDraft}
                      editorRef={editorRef}
                      factorList={factorList}
                      optionConfigMap={factorOptionsMap}
                      onCancel={() => onCancel(index)}
                      onDelete={() => onRuleDelete(index)}
                      onConfirm={onConfirm}
                      onDirtyChange={setDraftDirty}
                    />
                  ) : (
                    <div className="flex min-w-0 flex-col gap-4">
                      <div
                        className={cn(
                          "flex min-w-0 items-start",
                          reorderRules ? "justify-between gap-4" : "gap-2"
                        )}
                      >
                        <h2
                          className={cn(
                            "min-w-0 wrap-anywhere text-base font-semibold text-main",
                            reorderRules && "flex-1"
                          )}
                        >
                          {title}
                        </h2>
                        {canEdit &&
                          (reorderRules ? (
                            <div className="flex shrink-0 gap-1">
                              {[
                                [
                                  -1,
                                  ArrowUp,
                                  t(
                                    "settings.sensitive-data.global-rules.move-up"
                                  ),
                                ],
                                [
                                  1,
                                  ArrowDown,
                                  t(
                                    "settings.sensitive-data.global-rules.move-down"
                                  ),
                                ],
                              ].map(([offset, Icon, key]) => {
                                const DirectionIcon = Icon as typeof ArrowUp;
                                const direction = offset as number;
                                return (
                                  <Tooltip
                                    key={direction}
                                    content={key as string}
                                  >
                                    <Button
                                      appearance="secondary"
                                      size="sm"
                                      aria-label={key as string}
                                      disabled={
                                        processing ||
                                        draftPreparing ||
                                        index + direction < 0 ||
                                        index + direction >= items.length
                                      }
                                      onClick={() =>
                                        onReorder(index, direction)
                                      }
                                    >
                                      <DirectionIcon className="size-4" />
                                    </Button>
                                  </Tooltip>
                                );
                              })}
                            </div>
                          ) : (
                            <Tooltip
                              content={t(
                                "settings.sensitive-data.global-rules.edit-rule-tooltip"
                              )}
                              render={
                                <Button
                                  appearance="secondary"
                                  size="sm"
                                  className="shrink-0"
                                  aria-label={t(
                                    "settings.sensitive-data.global-rules.edit-rule",
                                    { name: title }
                                  )}
                                  disabled={
                                    processing ||
                                    draftPreparing ||
                                    activeEditor ||
                                    hasPendingQueue
                                  }
                                  onClick={() => onEdit(index)}
                                />
                              }
                            >
                              <Pencil className="size-4" />
                            </Tooltip>
                          ))}
                      </div>
                      <dl className="flex min-w-0 flex-col gap-4">
                        <div className="flex min-w-0 flex-col gap-2">
                          <dt className="text-xs text-control-light">
                            {t(
                              "settings.sensitive-data.global-rules.semantic-type"
                            )}
                          </dt>
                          <dd className="min-w-0">
                            {semanticType ? (
                              <SemanticTypeOption
                                semanticType={semanticType}
                                descriptionAsTooltip
                              />
                            ) : (
                              <span className="wrap-anywhere">
                                {item.rule.semanticType}
                              </span>
                            )}
                          </dd>
                        </div>
                        <div className="flex min-w-0 flex-col gap-2">
                          <dt className="text-xs text-control-light">
                            {t(
                              "settings.sensitive-data.global-rules.condition"
                            )}
                          </dt>
                          <dd className="min-w-0">
                            <MaskingRuleCondition
                              key={item.rule.condition?.expression ?? ""}
                              expression={item.rule.condition?.expression ?? ""}
                              factorList={factorList}
                              optionConfigMap={factorOptionsMap}
                            />
                          </dd>
                        </div>
                      </dl>
                    </div>
                  )}
                  {index < items.length - 1 && <Separator className="mt-4" />}
                </li>
              );
            })}
          </ol>
        )}
      </div>
    </WorkspacePageLayout>
  );
}
