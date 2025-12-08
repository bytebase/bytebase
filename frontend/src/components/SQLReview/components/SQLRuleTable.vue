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
        class="flex flex-col lg:hidden border px-2 pb-4 divide-y divide-block-border"
      >
        <div
          v-for="rule in category.ruleList"
          :key="rule.type"
          class="pt-4 flex flex-col gap-y-3"
        >
          <div class="flex justify-between items-center gap-x-2">
            <div class="flex items-center gap-x-1">
              <span>
                {{ getRuleLocalization(ruleTypeToString(rule.type), rule.engine).title }}
                <a
                  :href="`https://docs.bytebase.com/sql-review/review-rules#${rule.type}`"
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
            <div v-else class="flex items-center gap-x-2">
              <PencilIcon
                v-if="editable"
                class="w-4 h-4 cursor-pointer hover:text-accent"
                @click="setActiveRule(rule)"
              />
              <NButton
                v-if="editable"
                secondary
                type="error"
                size="small"
                @click="$emit('rule-remove', rule)"
              >
                {{ $t("common.delete") }}
              </NButton>
            </div>
          </div>
          <RuleLevelSwitch
            v-if="!hideLevel"
            class="text-xs"
            :level="rule.level"
            :disabled="!editable"
            @level-change="updateLevel(rule, $event)"
          />
          <p class="textinfolabel">
            {{ getRuleLocalization(ruleTypeToString(rule.type), rule.engine).description }}
          </p>
        </div>
      </div>
    </template>

    <SQLRuleEditDialog
      v-if="state.activeRule"
      :rule="state.activeRule"
      :disabled="!editable"
      @cancel="state.activeRule = undefined"
      @update:rule="onRuleChanged"
    />
  </div>
</template>

<script lang="tsx" setup>
import { ExternalLinkIcon, PencilIcon } from "lucide-vue-next";
import type { DataTableColumn } from "naive-ui";
import { NButton, NCheckbox, NDataTable, NDivider } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import type { RuleTemplateV2 } from "@/types";
import { getRuleLocalization, ruleTypeToString } from "@/types";
import { SQLReviewRule_Level } from "@/types/proto-es/v1/review_config_service_pb";
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
  (event: "rule-remove", rule: RuleTemplateV2): void;
  (event: "update:selectedRuleKeys", keys: string[]): void;
}>();

const { t } = useI18n();
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
          getRuleLocalization(ruleTypeToString(rule.type), rule.engine)
            .description || rule.componentList.length > 0
        );
      },
      renderExpand: (rule: RuleTemplateV2) => {
        const comment = getRuleLocalization(
          ruleTypeToString(rule.type),
          rule.engine
        ).description;
        return (
          <div class="px-10">
            <p class="w-full text-left text-gray-500">{comment}</p>
            {rule.componentList.length > 0 && !!comment && (
              <NDivider class={"my-4!"} />
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
      title: t("common.name"),
      resizable: true,
      key: "name",
      render: (rule: RuleTemplateV2) => {
        return (
          <div class="flex items-center gap-x-2">
            <span>
              {
                getRuleLocalization(ruleTypeToString(rule.type), rule.engine)
                  .title
              }
            </span>
            <a
              href={`https://docs.bytebase.com/sql-review/review-rules#${rule.type}`}
              target="_blank"
              class="flex flex-row gap-x-2 items-center text-base text-gray-500 hover:text-gray-900"
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
            disabled={!props.editable}
            onLevel-change={(on) => updateLevel(rule, on)}
          />
        );
      },
    },
    {
      title: t("common.operations"),
      hide: props.supportSelect,
      key: "operations",
      width: "12rem",
      render: (rule: RuleTemplateV2) => {
        return (
          <div class="flex items-center gap-x-2">
            <NButton onClick={() => setActiveRule(rule)}>
              {props.editable ? t("common.edit") : t("common.view")}
            </NButton>
            {props.editable && (
              <NButton
                secondary
                type="error"
                onClick={() => emit("rule-remove", rule)}
              >
                {t("common.delete")}
              </NButton>
            )}
          </div>
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

const setActiveRule = (rule: RuleTemplateV2) => {
  state.activeRule = rule;
};

const onRuleChanged = (update: Partial<RuleTemplateV2>) => {
  if (!state.activeRule) {
    return;
  }
  emit("rule-upsert", state.activeRule, update);
};

const updateLevel = (rule: RuleTemplateV2, level: SQLReviewRule_Level) => {
  emit("rule-upsert", rule, { level });
};
</script>
