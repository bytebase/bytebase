<template>
  <div>
    <template v-for="category in ruleList" :key="category.value">
      <div class="flex my-3 items-center">
        <span class="text-xl text-main font-semibold">
          {{ category.label }}
        </span>
        <span class="text-control-light text-md ml-1">
          ({{ category.ruleList.length }})
        </span>
      </div>
      <div class="hidden lg:block">
        <NDataTable
          key="sql-review-rule-table"
          :size="size"
          :striped="true"
          :columns="columns"
          :data="category.ruleList"
          :row-key="getRuleKey"
          :max-height="'100%'"
          :virtual-scroll="true"
          :bordered="true"
          :default-expand-all="true"
          :row-props="rowProps"
          :checked-row-keys="selectedRuleKeys"
          @update:checked-row-keys="
            (val) => $emit('update:selectedRuleKeys', val as string[])
          "
        />
      </div>
      <div
        class="flex flex-col lg:hidden border px-2 pb-4 divide-y space-y-4 divide-block-border"
      >
        <div
          v-for="rule in category.ruleList"
          :key="rule.type"
          class="pt-4 space-y-3"
        >
          <div class="flex justify-between items-center gap-x-2">
            <div class="flex items-center gap-x-1">
              <NTooltip
                v-if="!isRuleAvailable(rule)"
                trigger="hover"
                :show-arrow="false"
              >
                <template #trigger>
                  <div class="flex justify-center">
                    <heroicons-outline:exclamation
                      class="h-5 w-5 text-yellow-600"
                    />
                  </div>
                </template>
                <span class="whitespace-nowrap">
                  {{
                    $t("sql-review.not-available-for-free", {
                      plan: $t(
                        `subscription.plan.${planTypeToString(
                          currentPlan
                        )}.title`
                      ),
                    })
                  }}
                </span>
              </NTooltip>
              <span>
                {{ getRuleLocalization(rule.type, rule.engine).title }}
                <a
                  :href="`https://www.bytebase.com/docs/sql-review/review-rules#${rule.type}`"
                  target="_blank"
                  class="inline-block"
                >
                  <ExternalLinkIcon class="w-4 h-4" />
                </a>
              </span>
            </div>
            <NCheckbox
              v-if="supportSelect"
              :checked="selectedRuleKeys.includes(getRuleKey(rule))"
              @update:checked="(_) => toggleRule(rule)"
            />
            <div v-else class="flex items-center space-x-2">
              <PencilIcon
                v-if="editable"
                class="w-4 h-4"
                @click="setActiveRule(rule)"
              />
              <NSwitch
                size="small"
                :disabled="!editable || !isRuleAvailable(rule)"
                :value="rule.level !== SQLReviewRuleLevel.DISABLED"
                @update-value="(val) => toggleActivity(rule, val)"
              />
            </div>
          </div>
          <RuleLevelSwitch
            v-if="!hideLevel"
            class="text-xs"
            :level="rule.level"
            :disabled="!editable || !isRuleAvailable(rule)"
            @level-change="updateLevel(rule, $event)"
          />
          <p class="textinfolabel">
            {{ getRuleLocalization(rule.type, rule.engine).description }}
          </p>
        </div>
      </div>
    </template>

    <SQLRuleEditDialog
      v-if="state.activeRule"
      :rule="state.activeRule"
      :disabled="!editable || !isRuleAvailable(state.activeRule)"
      @cancel="state.activeRule = undefined"
      @update:rule="onRuleChanged"
    />
  </div>
</template>

<script lang="tsx" setup>
import { ExternalLinkIcon, PencilIcon } from "lucide-vue-next";
import {
  NSwitch,
  NCheckbox,
  NDataTable,
  NButton,
  NDivider,
  NTooltip,
} from "naive-ui";
import type { DataTableColumn } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useCurrentPlan } from "@/store";
import type { RuleTemplateV2 } from "@/types";
import {
  getRuleLocalization,
  ruleIsAvailableInSubscription,
  planTypeToString,
} from "@/types";
import { SQLReviewRuleLevel } from "@/types/proto/v1/org_policy_service";
import RuleConfig from "./RuleConfigComponents/RuleConfig.vue";
import RuleLevelSwitch from "./RuleLevelSwitch.vue";
import type { RuleListWithCategory } from "./SQLReviewCategoryTabFilter.vue";
import SQLRuleEditDialog from "./SQLRuleEditDialog.vue";
import { getRuleKey } from "./utils";

type LocalState = {
  activeRule: RuleTemplateV2 | undefined;
};

const props = withDefaults(
  defineProps<{
    ruleList?: RuleListWithCategory[];
    editable: boolean;
    supportSelect?: boolean;
    hideLevel?: boolean;
    selectedRuleKeys?: string[];
    size?: "small" | "medium";
  }>(),
  {
    ruleList: () => [],
    editable: true,
    supportSelect: false,
    hideLevel: false,
    selectedRuleKeys: () => [],
    size: "medium",
  }
);

const emit = defineEmits<{
  (
    event: "rule-upsert",
    rule: RuleTemplateV2,
    update: Partial<RuleTemplateV2>
  ): void;
  (event: "update:selectedRuleKeys", keys: string[]): void;
}>();

const { t } = useI18n();
const currentPlan = useCurrentPlan();
const state = reactive<LocalState>({
  activeRule: undefined,
});

const columns = computed(() => {
  const columns: (DataTableColumn<RuleTemplateV2> & { hide?: boolean })[] = [
    {
      type: "selection",
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
      hide: !props.supportSelect,
    },
    {
      type: "expand",
      expandable: (rule: RuleTemplateV2) => {
        return !!(
          rule.comment ||
          getRuleLocalization(rule.type, rule.engine).description ||
          rule.componentList.length > 0
        );
      },
      renderExpand: (rule: RuleTemplateV2) => {
        const comment =
          rule.comment ||
          getRuleLocalization(rule.type, rule.engine).description;
        return (
          <div class="px-10">
            <p class="w-full text-left text-gray-500">{comment}</p>
            {rule.componentList.length > 0 && !!comment && (
              <NDivider class={"!my-4"} />
            )}
            {rule.componentList.length > 0 && (
              <RuleConfig
                disabled={true}
                rule={rule}
                size={"small"}
                class={"mb-3"}
              />
            )}
          </div>
        );
      },
      cellProps: () => {
        return {
          onClick: (e: MouseEvent) => {
            e.stopPropagation();
          },
        };
      },
    },
    {
      title: t("sql-review.rule.active"),
      key: "active",
      width: "5rem",
      hide: props.hideLevel,
      render: (rule: RuleTemplateV2) => {
        return (
          <NSwitch
            size={"small"}
            disabled={!props.editable || !isRuleAvailable(rule)}
            value={rule.level !== SQLReviewRuleLevel.DISABLED}
            onUpdate:value={(val) => toggleActivity(rule, val)}
          />
        );
      },
    },
    {
      title: t("common.name"),
      resizable: true,
      key: "name",
      render: (rule: RuleTemplateV2) => {
        return (
          <div class="flex items-center space-x-2">
            <span>{getRuleLocalization(rule.type, rule.engine).title}</span>
            <a
              href={`https://www.bytebase.com/docs/sql-review/review-rules#${rule.type}`}
              target="_blank"
              class="flex flex-row space-x-2 items-center text-base text-gray-500 hover:text-gray-900"
            >
              <ExternalLinkIcon class="w-4 h-4" />
            </a>
          </div>
        );
      },
    },
    {
      title: t("sql-review.level.name"),
      hide: props.hideLevel,
      key: "level",
      render: (rule: RuleTemplateV2) => {
        return (
          <RuleLevelSwitch
            level={rule.level}
            disabled={!props.editable || !isRuleAvailable(rule)}
            onLevel-change={(on) => updateLevel(rule, on)}
          />
        );
      },
    },
    {
      title: t("common.operations"),
      hide: props.supportSelect,
      key: "operations",
      width: "10rem",
      render: (rule: RuleTemplateV2) => {
        return (
          <NButton
            disabled={!isRuleAvailable(rule)}
            onClick={() => setActiveRule(rule)}
          >
            {props.editable ? t("common.edit") : t("common.view")}
          </NButton>
        );
      },
    },
  ];

  return columns.filter((item) => !item.hide);
});

const rowProps = (rule: RuleTemplateV2) => {
  return {
    style: props.supportSelect ? "cursor: pointer;" : "",
    onClick: () => {
      if (props.supportSelect) {
        toggleRule(rule);
        return;
      }
    },
  };
};

const toggleRule = (rule: RuleTemplateV2) => {
  const key = getRuleKey(rule);
  const index = props.selectedRuleKeys.findIndex((k) => k === key);
  if (index < 0) {
    emit("update:selectedRuleKeys", [...props.selectedRuleKeys, key]);
  } else {
    emit("update:selectedRuleKeys", [
      ...props.selectedRuleKeys.slice(0, index),
      ...props.selectedRuleKeys.slice(index + 1),
    ]);
  }
};

const isRuleAvailable = (rule: RuleTemplateV2) => {
  return ruleIsAvailableInSubscription(rule.type, currentPlan.value);
};

const setActiveRule = (rule: RuleTemplateV2) => {
  state.activeRule = rule;
};

const toggleActivity = (rule: RuleTemplateV2, on: boolean) => {
  updateLevel(
    rule,
    on ? SQLReviewRuleLevel.WARNING : SQLReviewRuleLevel.DISABLED
  );
};

const onRuleChanged = (update: Partial<RuleTemplateV2>) => {
  if (!state.activeRule) {
    return;
  }
  emit("rule-upsert", state.activeRule, update);
};

const updateLevel = (rule: RuleTemplateV2, level: SQLReviewRuleLevel) => {
  emit("rule-upsert", rule, { level });
};
</script>
