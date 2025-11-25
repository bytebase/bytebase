# Approval Rules Frontend Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Redesign Custom Approval UI to support direct approval rules (source + CEL condition → inline approval flow) without risk level abstraction.

**Architecture:** Replace the source×level grid with a source-grouped table of rules. Each rule has inline approval flow definition (no templates). Reuse ExprEditor for conditions, StepsTable for approval flow editing.

**Tech Stack:** Vue 3, Naive UI, Pinia, proto-es, CEL

---

## Task 1: Update Type Definitions

**Files:**
- Modify: `frontend/src/types/workspaceApprovalSetting.ts`

**Step 1: Replace types with new structure**

Replace the entire file contents:

```typescript
import type { ConditionGroupExpr } from "@/plugins/cel";
import type { ApprovalFlow } from "@/types/proto-es/v1/issue_service_pb";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";

// A single approval rule with inline flow definition
export type LocalApprovalRule = {
  uid: string; // Local unique identifier for UI tracking
  source: WorkspaceApprovalSetting_Rule_Source;
  condition: string; // CEL expression string
  conditionExpr?: ConditionGroupExpr; // Parsed CEL for editor
  flow: ApprovalFlow; // Inline approval flow (roles array)
};

// The local config is just a list of rules per source
export type LocalApprovalConfig = {
  rules: LocalApprovalRule[];
};
```

**Step 2: Verify no type errors**

Run: `pnpm --dir frontend type-check`
Expected: Type errors in files that import old types (expected, will fix in subsequent tasks)

---

## Task 2: Update Utility Functions

**Files:**
- Modify: `frontend/src/utils/workspaceApprovalSetting.ts`

**Step 1: Rewrite resolveLocalApprovalConfig**

Replace the entire file with simplified logic:

```typescript
import { create } from "@bufbuild/protobuf";
import { v4 as uuidv4 } from "uuid";
import type { ConditionGroupExpr } from "@/plugins/cel";
import { resolveCELExpr, wrapAsGroup, emptySimpleExpr } from "@/plugins/cel";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import { ExprSchema as CELExprSchema } from "@/types/proto-es/google/api/expr/v1alpha1/syntax_pb";
import { ExprSchema } from "@/types/proto-es/google/type/expr_pb";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { WorkspaceApprovalSetting } from "@/types/proto-es/v1/setting_service_pb";
import {
  WorkspaceApprovalSetting_RuleSchema,
  WorkspaceApprovalSettingSchema,
  WorkspaceApprovalSetting_Rule_Source,
} from "@/types/proto-es/v1/setting_service_pb";
import { batchConvertCELStringToParsedExpr, batchConvertParsedExprToCELString } from "@/utils";
import { displayRoleTitle } from "./role";

export const approvalNodeRoleText = (role: string) => {
  return displayRoleTitle(role);
};

// Convert proto WorkspaceApprovalSetting to local format
export const resolveLocalApprovalConfig = async (
  config: WorkspaceApprovalSetting
): Promise<LocalApprovalConfig> => {
  const rules: LocalApprovalRule[] = [];
  const expressions: string[] = [];
  const ruleIndices: number[] = [];

  for (let i = 0; i < config.rules.length; i++) {
    const protoRule = config.rules[i];
    const condition = protoRule.condition?.expression || "";

    const rule: LocalApprovalRule = {
      uid: uuidv4(),
      source: protoRule.source || WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED,
      condition,
      flow: protoRule.template?.flow || create(ApprovalFlowSchema, { roles: [] }),
    };
    rules.push(rule);

    if (condition) {
      expressions.push(condition);
      ruleIndices.push(i);
    }
  }

  // Parse CEL expressions in batch
  if (expressions.length > 0) {
    const parsedExprs = await batchConvertCELStringToParsedExpr(expressions);
    for (let i = 0; i < parsedExprs.length; i++) {
      const ruleIndex = ruleIndices[i];
      const parsed = parsedExprs[i];
      if (parsed) {
        rules[ruleIndex].conditionExpr = wrapAsGroup(
          resolveCELExpr(parsed)
        ) as ConditionGroupExpr;
      }
    }
  }

  return { rules };
};

// Convert local format back to proto WorkspaceApprovalSetting
export const buildWorkspaceApprovalSetting = async (
  config: LocalApprovalConfig
): Promise<WorkspaceApprovalSetting> => {
  const protoRules = [];

  for (const rule of config.rules) {
    const protoRule = create(WorkspaceApprovalSetting_RuleSchema, {
      source: rule.source,
      condition: create(ExprSchema, { expression: rule.condition }),
      template: {
        flow: rule.flow,
      },
    });
    protoRules.push(protoRule);
  }

  return create(WorkspaceApprovalSettingSchema, {
    rules: protoRules,
  });
};

// Helper: Get rules filtered by source
export const getRulesBySource = (
  config: LocalApprovalConfig,
  source: WorkspaceApprovalSetting_Rule_Source
): LocalApprovalRule[] => {
  return config.rules.filter((r) => r.source === source);
};

// Helper: Format approval flow for display
export const formatApprovalFlow = (flow: ApprovalFlow): string => {
  if (!flow.roles || flow.roles.length === 0) {
    return "-";
  }
  return flow.roles.map((r) => displayRoleTitle(r)).join(" → ");
};
```

**Step 2: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 3: Update Store

**Files:**
- Modify: `frontend/src/store/modules/workspaceApprovalSetting.ts`

**Step 1: Simplify store to work with new format**

Replace the entire file:

```typescript
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { defineStore } from "pinia";
import { v4 as uuidv4 } from "uuid";
import { ref } from "vue";
import { settingServiceClientConnect } from "@/grpcweb";
import type { LocalApprovalConfig, LocalApprovalRule } from "@/types";
import { ApprovalFlowSchema } from "@/types/proto-es/v1/issue_service_pb";
import type { Setting } from "@/types/proto-es/v1/setting_service_pb";
import {
  GetSettingRequestSchema,
  Setting_SettingName,
  SettingSchema,
  ValueSchema as SettingValueSchema,
  UpdateSettingRequestSchema,
  WorkspaceApprovalSetting_Rule_Source,
} from "@/types/proto-es/v1/setting_service_pb";
import {
  buildWorkspaceApprovalSetting,
  resolveLocalApprovalConfig,
} from "@/utils";
import { useGracefulRequest } from "./utils";

const SETTING_NAME = `settings/${Setting_SettingName[Setting_SettingName.WORKSPACE_APPROVAL]}`;

export const useWorkspaceApprovalSettingStore = defineStore(
  "workspaceApprovalSetting",
  () => {
    const config = ref<LocalApprovalConfig>({
      rules: [],
    });

    const setConfigSetting = async (setting: Setting) => {
      if (setting.value?.value?.case === "workspaceApprovalSettingValue") {
        const _config = setting.value.value.value;
        config.value = await resolveLocalApprovalConfig(_config);
      }
    };

    const fetchConfig = async () => {
      try {
        const request = create(GetSettingRequestSchema, {
          name: SETTING_NAME,
        });
        const response = await settingServiceClientConnect.getSetting(request);
        await setConfigSetting(response);
      } catch (ex) {
        console.error(ex);
      }
    };

    const updateConfig = async () => {
      const approvalSetting = await buildWorkspaceApprovalSetting(config.value);

      const setting = create(SettingSchema, {
        name: SETTING_NAME,
        value: create(SettingValueSchema, {
          value: {
            case: "workspaceApprovalSettingValue",
            value: approvalSetting,
          },
        }),
      });

      const request = create(UpdateSettingRequestSchema, {
        allowMissing: true,
        setting,
      });
      await settingServiceClientConnect.updateSetting(request);
    };

    const useBackupAndUpdateConfig = async (update: () => Promise<void>) => {
      const backup = cloneDeep(config.value);
      try {
        await useGracefulRequest(update);
      } catch (err) {
        config.value = backup;
        throw err;
      }
    };

    // Get rules for a specific source
    const getRulesBySource = (source: WorkspaceApprovalSetting_Rule_Source): LocalApprovalRule[] => {
      return config.value.rules.filter((r) => r.source === source);
    };

    // Add a new rule
    const addRule = async (rule: Omit<LocalApprovalRule, "uid">) => {
      await useBackupAndUpdateConfig(async () => {
        const newRule: LocalApprovalRule = {
          ...rule,
          uid: uuidv4(),
        };
        config.value.rules.push(newRule);
        await updateConfig();
      });
    };

    // Update an existing rule
    const updateRule = async (uid: string, updates: Partial<LocalApprovalRule>) => {
      await useBackupAndUpdateConfig(async () => {
        const index = config.value.rules.findIndex((r) => r.uid === uid);
        if (index >= 0) {
          config.value.rules[index] = {
            ...config.value.rules[index],
            ...updates,
          };
          await updateConfig();
        }
      });
    };

    // Delete a rule
    const deleteRule = async (uid: string) => {
      await useBackupAndUpdateConfig(async () => {
        const index = config.value.rules.findIndex((r) => r.uid === uid);
        if (index >= 0) {
          config.value.rules.splice(index, 1);
          await updateConfig();
        }
      });
    };

    // Reorder rules (for drag-and-drop within a source)
    const reorderRules = async (source: WorkspaceApprovalSetting_Rule_Source, fromIndex: number, toIndex: number) => {
      await useBackupAndUpdateConfig(async () => {
        // Get all rules for this source
        const sourceRules = config.value.rules.filter((r) => r.source === source);
        const otherRules = config.value.rules.filter((r) => r.source !== source);

        // Reorder within source rules
        const [moved] = sourceRules.splice(fromIndex, 1);
        sourceRules.splice(toIndex, 0, moved);

        // Rebuild config with reordered rules
        config.value.rules = [...otherRules, ...sourceRules];
        await updateConfig();
      });
    };

    return {
      config,
      fetchConfig,
      getRulesBySource,
      addRule,
      updateRule,
      deleteRule,
      reorderRules,
    };
  }
);
```

**Step 2: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 4: Create RuleEditModal Component

**Files:**
- Create: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/RulesPanel/RuleEditModal.vue`

**Step 1: Create the modal component**

```vue
<template>
  <NModal v-model:show="show" preset="dialog" :title="modalTitle" :show-icon="false">
    <div class="w-[60vw] max-w-4xl flex flex-col gap-y-4">
      <!-- Condition Section -->
      <div class="flex flex-col gap-y-2">
        <h3 class="font-medium text-sm text-control">
          {{ $t("cel.condition.self") }}
        </h3>
        <div class="text-sm text-control-light mb-2">
          {{ $t("cel.condition.description-tips") }}
        </div>
        <ExprEditor
          :expr="state.conditionExpr"
          :allow-admin="true"
          :factor-list="factorList"
          :option-config-map="optionConfigMap"
          @update="handleConditionUpdate"
        />
      </div>

      <!-- Approval Flow Section -->
      <div class="flex flex-col gap-y-2">
        <h3 class="font-medium text-sm text-control">
          {{ $t("custom-approval.approval-flow.self") }}
        </h3>
        <StepsTable :flow="state.flow" :editable="true" @update="handleFlowUpdate" />
      </div>
    </div>

    <template #action>
      <div class="flex justify-end gap-x-2">
        <NButton @click="handleCancel">{{ $t("common.cancel") }}</NButton>
        <NButton type="primary" :disabled="!canSave" @click="handleSave">
          {{ mode === "create" ? $t("common.add") : $t("common.update") }}
        </NButton>
      </div>
    </template>
  </NModal>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { NButton, NModal } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import ExprEditor from "@/components/ExprEditor";
import type { ConditionGroupExpr } from "@/plugins/cel";
import {
  buildCELExpr,
  emptySimpleExpr,
  validateSimpleExpr,
  wrapAsGroup,
} from "@/plugins/cel";
import type { LocalApprovalRule } from "@/types";
import { ApprovalFlowSchema, type ApprovalFlow } from "@/types/proto-es/v1/issue_service_pb";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { batchConvertParsedExprToCELString } from "@/utils";
import { getFactorList, getOptionConfigMap } from "../../common/utils";
import StepsTable from "../common/StepsTable.vue";

type LocalState = {
  conditionExpr: ConditionGroupExpr;
  flow: ApprovalFlow;
};

const props = defineProps<{
  show: boolean;
  mode: "create" | "edit";
  source: WorkspaceApprovalSetting_Rule_Source;
  rule?: LocalApprovalRule;
}>();

const emit = defineEmits<{
  (event: "update:show", value: boolean): void;
  (event: "save", rule: Omit<LocalApprovalRule, "uid">): void;
}>();

const { t } = useI18n();

const state = reactive<LocalState>({
  conditionExpr: wrapAsGroup(emptySimpleExpr()),
  flow: create(ApprovalFlowSchema, { roles: [] }),
});

const show = computed({
  get: () => props.show,
  set: (v) => emit("update:show", v),
});

const modalTitle = computed(() => {
  return props.mode === "create"
    ? t("custom-approval.approval-flow.add-rule")
    : t("custom-approval.approval-flow.edit-rule");
});

const factorList = computed(() => getFactorList(props.source));
const optionConfigMap = computed(() => getOptionConfigMap(props.source));

const canSave = computed(() => {
  // Must have valid condition
  if (!validateSimpleExpr(state.conditionExpr)) {
    return false;
  }
  // Must have at least one approver
  if (!state.flow.roles || state.flow.roles.length === 0) {
    return false;
  }
  return true;
});

const handleConditionUpdate = () => {
  // Condition updated via ExprEditor - state.conditionExpr is reactive
};

const handleFlowUpdate = () => {
  // Flow updated via StepsTable - state.flow is reactive
};

const handleCancel = () => {
  show.value = false;
};

const handleSave = async () => {
  // Build CEL expression string from conditionExpr
  const celExpr = await buildCELExpr(state.conditionExpr);
  let conditionString = "";
  if (celExpr) {
    const expressions = await batchConvertParsedExprToCELString([celExpr]);
    conditionString = expressions[0] || "";
  }

  emit("save", {
    source: props.source,
    condition: conditionString,
    conditionExpr: cloneDeep(state.conditionExpr),
    flow: cloneDeep(state.flow),
  });

  show.value = false;
};

// Initialize state when rule changes
watch(
  () => props.rule,
  (rule) => {
    if (rule) {
      state.conditionExpr = rule.conditionExpr
        ? cloneDeep(rule.conditionExpr)
        : wrapAsGroup(emptySimpleExpr());
      state.flow = rule.flow
        ? cloneDeep(rule.flow)
        : create(ApprovalFlowSchema, { roles: [] });
    } else {
      state.conditionExpr = wrapAsGroup(emptySimpleExpr());
      state.flow = create(ApprovalFlowSchema, { roles: [] });
    }
  },
  { immediate: true }
);
</script>
```

**Step 2: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 5: Update RulesSection Component

**Files:**
- Modify: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/RulesPanel/RulesSection.vue`

**Step 1: Replace with new table-based layout**

```vue
<template>
  <div class="flex flex-col gap-y-2">
    <div class="flex items-center justify-between">
      <div class="font-medium text-base">
        {{ sourceText(source) }}
      </div>
      <NTooltip>
        <template #trigger>
          <HelpCircleIcon class="w-4 h-4 text-control-light cursor-help" />
        </template>
        {{ $t("custom-approval.rule.first-match-wins") }}
      </NTooltip>
    </div>

    <NDataTable
      size="small"
      :columns="columns"
      :data="rules"
      :striped="true"
      :bordered="true"
      :row-key="(row: LocalApprovalRule) => row.uid"
    />

    <div class="mt-2">
      <NButton size="small" @click="handleAddRule">
        <template #icon>
          <PlusIcon class="w-4" />
        </template>
        {{ $t("custom-approval.approval-flow.add-rule") }}
      </NButton>
    </div>

    <RuleEditModal
      v-model:show="showModal"
      :mode="modalMode"
      :source="source"
      :rule="editingRule"
      @save="handleSaveRule"
    />
  </div>
</template>

<script lang="tsx" setup>
import { HelpCircleIcon, PencilIcon, PlusIcon, TrashIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NButton, NDataTable, NTooltip, NPopconfirm } from "naive-ui";
import { computed, ref } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useWorkspaceApprovalSettingStore } from "@/store";
import type { LocalApprovalRule } from "@/types";
import type { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import { formatApprovalFlow } from "@/utils";
import { sourceText } from "../../common";
import { useCustomApprovalContext } from "../context";
import RuleEditModal from "./RuleEditModal.vue";

const props = defineProps<{
  source: WorkspaceApprovalSetting_Rule_Source;
}>();

const { t } = useI18n();
const store = useWorkspaceApprovalSettingStore();
const context = useCustomApprovalContext();

const showModal = ref(false);
const modalMode = ref<"create" | "edit">("create");
const editingRule = ref<LocalApprovalRule | undefined>();

const rules = computed(() => store.getRulesBySource(props.source));

const columns = computed((): DataTableColumn<LocalApprovalRule>[] => [
  {
    title: t("cel.condition.self"),
    key: "condition",
    ellipsis: { tooltip: true },
    render: (row) => (
      <code class="text-xs bg-control-bg px-1 py-0.5 rounded">
        {row.condition || "true"}
      </code>
    ),
  },
  {
    title: t("custom-approval.approval-flow.self"),
    key: "flow",
    width: 280,
    render: (row) => formatApprovalFlow(row.flow),
  },
  {
    title: t("common.operations"),
    key: "operations",
    width: 100,
    render: (row) => (
      <div class="flex gap-x-1">
        <NButton size="tiny" onClick={() => handleEditRule(row)}>
          <PencilIcon class="w-3" />
        </NButton>
        <NPopconfirm onPositiveClick={() => handleDeleteRule(row)}>
          {{
            trigger: () => (
              <NButton size="tiny">
                <TrashIcon class="w-3" />
              </NButton>
            ),
            default: () => t("common.delete-confirm"),
          }}
        </NPopconfirm>
      </div>
    ),
  },
]);

const handleAddRule = () => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }
  modalMode.value = "create";
  editingRule.value = undefined;
  showModal.value = true;
};

const handleEditRule = (rule: LocalApprovalRule) => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }
  modalMode.value = "edit";
  editingRule.value = rule;
  showModal.value = true;
};

const handleDeleteRule = async (rule: LocalApprovalRule) => {
  if (!context.hasFeature.value) {
    context.showFeatureModal.value = true;
    return;
  }
  try {
    await store.deleteRule(rule.uid);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.deleted"),
    });
  } catch {
    // Error handled by store
  }
};

const handleSaveRule = async (ruleData: Omit<LocalApprovalRule, "uid">) => {
  try {
    if (modalMode.value === "create") {
      await store.addRule(ruleData);
    } else if (editingRule.value) {
      await store.updateRule(editingRule.value.uid, ruleData);
    }
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } catch {
    // Error handled by store
  }
};
</script>
```

**Step 2: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 6: Update RulesPanel Component

**Files:**
- Modify: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/RulesPanel/RulesPanel.vue`

**Step 1: Update to use new Source enum**

```vue
<template>
  <div class="w-full flex flex-col gap-y-6">
    <RulesSection
      v-for="source in supportedSourceList"
      :key="source"
      :source="source"
    />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";
import RulesSection from "./RulesSection.vue";

// Use the new approval rule source enum directly
const supportedSourceList = computed(() => [
  WorkspaceApprovalSetting_Rule_Source.DDL,
  WorkspaceApprovalSetting_Rule_Source.DML,
  WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE,
  WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA,
  WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE,
]);
</script>
```

**Step 2: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 7: Update CustomApproval Component (Remove Tabs)

**Files:**
- Modify: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/CustomApproval.vue`

**Step 1: Remove tabs, show only RulesPanel**

```vue
<template>
  <div class="w-full">
    <RulesPanel />
  </div>
</template>

<script lang="ts" setup>
import { provideRiskFilter } from "../common/RiskFilter";
import RulesPanel from "./RulesPanel";

provideRiskFilter();
</script>
```

**Step 2: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 8: Update sourceText Helper

**Files:**
- Modify: `frontend/src/components/CustomApproval/Settings/components/common/utils.ts`

**Step 1: Add sourceText overload for new Source enum**

Add this function near the existing `sourceText`:

```typescript
import { WorkspaceApprovalSetting_Rule_Source } from "@/types/proto-es/v1/setting_service_pb";

// Overload for new approval rule source enum
export const approvalSourceText = (source: WorkspaceApprovalSetting_Rule_Source) => {
  switch (source) {
    case WorkspaceApprovalSetting_Rule_Source.SOURCE_UNSPECIFIED:
      return t("common.all");
    case WorkspaceApprovalSetting_Rule_Source.DDL:
      return t("custom-approval.risk-rule.risk.namespace.ddl");
    case WorkspaceApprovalSetting_Rule_Source.DML:
      return t("custom-approval.risk-rule.risk.namespace.dml");
    case WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE:
      return t("custom-approval.risk-rule.risk.namespace.create_database");
    case WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA:
      return t("custom-approval.risk-rule.risk.namespace.data_export");
    case WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE:
      return t("custom-approval.risk-rule.risk.namespace.request-role");
    default:
      return "UNRECOGNIZED";
  }
};

// Map between Risk_Source and WorkspaceApprovalSetting_Rule_Source
export const ApprovalSourceFactorMap: Map<WorkspaceApprovalSetting_Rule_Source, Factor[]> = new Map([
  [WorkspaceApprovalSetting_Rule_Source.DDL, RiskSourceFactorMap.get(Risk_Source.DDL) || []],
  [WorkspaceApprovalSetting_Rule_Source.DML, RiskSourceFactorMap.get(Risk_Source.DML) || []],
  [WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE, RiskSourceFactorMap.get(Risk_Source.CREATE_DATABASE) || []],
  [WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA, RiskSourceFactorMap.get(Risk_Source.DATA_EXPORT) || []],
  [WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE, RiskSourceFactorMap.get(Risk_Source.REQUEST_ROLE) || []],
]);

export const getApprovalFactorList = (source: WorkspaceApprovalSetting_Rule_Source) => {
  return ApprovalSourceFactorMap.get(source) ?? [];
};

export const getApprovalOptionConfigMap = (source: WorkspaceApprovalSetting_Rule_Source) => {
  // Map to Risk_Source for reusing existing option config logic
  const riskSource = approvalSourceToRiskSource(source);
  return getOptionConfigMap(riskSource);
};

const approvalSourceToRiskSource = (source: WorkspaceApprovalSetting_Rule_Source): Risk_Source => {
  switch (source) {
    case WorkspaceApprovalSetting_Rule_Source.DDL:
      return Risk_Source.DDL;
    case WorkspaceApprovalSetting_Rule_Source.DML:
      return Risk_Source.DML;
    case WorkspaceApprovalSetting_Rule_Source.CREATE_DATABASE:
      return Risk_Source.CREATE_DATABASE;
    case WorkspaceApprovalSetting_Rule_Source.EXPORT_DATA:
      return Risk_Source.DATA_EXPORT;
    case WorkspaceApprovalSetting_Rule_Source.REQUEST_ROLE:
      return Risk_Source.REQUEST_ROLE;
    default:
      return Risk_Source.SOURCE_UNSPECIFIED;
  }
};
```

**Step 2: Update RulesSection to use new helpers**

Update imports in RulesSection.vue to use `approvalSourceText` and `getApprovalFactorList`.

**Step 3: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 9: Add i18n Keys

**Files:**
- Modify: `frontend/src/locales/en-US.json`
- Modify: `frontend/src/locales/zh-CN.json`

**Step 1: Add missing i18n keys for en-US**

Add under `custom-approval`:

```json
{
  "custom-approval": {
    "rule": {
      "first-match-wins": "Rules are evaluated in order. First matching rule applies."
    },
    "approval-flow": {
      "add-rule": "Add rule",
      "edit-rule": "Edit rule"
    }
  }
}
```

**Step 2: Add corresponding zh-CN translations**

```json
{
  "custom-approval": {
    "rule": {
      "first-match-wins": "规则按顺序评估，第一个匹配的规则将被应用。"
    },
    "approval-flow": {
      "add-rule": "添加规则",
      "edit-rule": "编辑规则"
    }
  }
}
```

**Step 3: Verify no missing keys**

Run: `pnpm --dir frontend dev` and check console for i18n warnings.

---

## Task 10: Remove Unused Components

**Files:**
- Delete: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/FlowsPanel/FlowsPanel.vue`
- Delete: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/FlowsPanel/FlowTable.vue`
- Delete: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/FlowsPanel/Toolbar.vue`
- Delete: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/RulesPanel/RuleSelect.vue`
- Delete: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/RulesPanel/RiskTips.vue`
- Delete: `frontend/src/components/CustomApproval/Settings/components/CustomApproval/RuleDialog/` (entire directory)

**Step 1: Remove FlowsPanel directory**

```bash
rm -rf frontend/src/components/CustomApproval/Settings/components/CustomApproval/FlowsPanel
```

**Step 2: Remove old RuleDialog directory**

```bash
rm -rf frontend/src/components/CustomApproval/Settings/components/CustomApproval/RuleDialog
```

**Step 3: Remove RuleSelect and RiskTips**

```bash
rm frontend/src/components/CustomApproval/Settings/components/CustomApproval/RulesPanel/RuleSelect.vue
rm frontend/src/components/CustomApproval/Settings/components/CustomApproval/RulesPanel/RiskTips.vue
```

**Step 4: Update FlowsPanel index.ts if exists**

Check and remove any barrel exports.

**Step 5: Verify no import errors**

Run: `pnpm --dir frontend type-check`

---

## Task 11: Update Type Exports

**Files:**
- Modify: `frontend/src/types/index.ts`

**Step 1: Update exports**

Ensure the new types are exported. Remove exports of deleted types like `ParsedApprovalRule`, `UnrecognizedApprovalRule`.

**Step 2: Verify compilation**

Run: `pnpm --dir frontend type-check`

---

## Task 12: Final Integration Test

**Step 1: Build frontend**

Run: `pnpm --dir frontend build`
Expected: Build succeeds with no errors

**Step 2: Run type check**

Run: `pnpm --dir frontend type-check`
Expected: No type errors

**Step 3: Run linter**

Run: `pnpm --dir frontend lint`
Expected: No lint errors (or only pre-existing ones)

**Step 4: Manual testing**

1. Start dev server: `pnpm --dir frontend dev`
2. Navigate to Custom Approval settings
3. Verify:
   - Rules are displayed grouped by source (DDL, DML, etc.)
   - Each source section shows a table with Condition | Approval Flow | Actions
   - "Add rule" button opens modal with condition builder + flow editor
   - Editing a rule works correctly
   - Deleting a rule works with confirmation
   - No "Flows" tab visible
