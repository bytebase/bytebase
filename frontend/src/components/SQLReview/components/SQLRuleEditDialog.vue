<template>
  <BBModal
    :title="getRuleLocalization(rule.type, rule.engine).title"
    @close="$emit('cancel')"
  >
    <div class="space-y-4 w-[calc(100vw-5rem)] sm:w-[40rem] pb-1">
      <div class="space-y-1">
        <div class="flex items-center space-x-2">
          <h3 class="text-lg text-control font-medium">
            {{ $t("common.name") }}
          </h3>
          <div class="flex items-center space-x-2">
            <RichEngineName
              :engine="rule.engine"
              tag="p"
              class="text-center text-sm !text-main"
            />
          </div>
        </div>
        <div class="textinfolabel flex items-center gap-x-2">
          {{ getRuleLocalization(rule.type, rule.engine).title }}
          <a
            :href="`https://docs.bytebase.com/sql-review/review-rules#${rule.type}`"
            target="__blank"
            class="flex flex-row space-x-2 items-center text-base text-gray-500 hover:text-gray-900"
          >
            <heroicons-outline:external-link class="w-4 h-4" />
          </a>
        </div>
      </div>
      <div class="space-y-1">
        <h3 class="text-lg text-control font-medium">
          {{ $t("sql-review.rule.active") }}
        </h3>
        <div class="flex items-center gap-x-2 text-sm">
          <NSwitch
            :disabled="disabled"
            :value="state.level !== SQLReviewRuleLevel.DISABLED"
            @update-value="toggleActivity"
          />
        </div>
      </div>
      <div class="space-y-1">
        <h3 class="text-lg text-control font-medium">
          {{ $t("sql-review.level.name") }}
        </h3>
        <div class="flex items-center gap-x-2 text-sm">
          <RuleLevelSwitch
            :level="state.level"
            :disabled="disabled"
            @level-change="state.level = $event"
          />
        </div>
      </div>
      <div class="space-y-1">
        <h3 class="text-lg text-control font-medium">
          {{ $t("common.description") }}
        </h3>
        <div class="flex flex-col gap-x-2">
          <BBTextField
            v-model:value="state.comment"
            :disabled="disabled"
            :placeholder="
              getRuleLocalization(rule.type, rule.engine).description ||
              $t('common.description')
            "
            type="textarea"
            :autosize="{ minRows: 1, maxRows: 4 }"
          />
        </div>
      </div>
      <RuleConfig
        ref="ruleConfig"
        :disabled="disabled"
        :rule="rule"
        :size="'medium'"
      />
      <div class="mt-4 pt-2 border-t flex justify-end space-x-3">
        <NButton @click.prevent="$emit('cancel')">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton type="primary" :disabled="disabled" @click.prevent="confirm">
          {{ $t("common.confirm") }}
        </NButton>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NButton, NSwitch } from "naive-ui";
import { nextTick, reactive, watch, ref } from "vue";
import { BBModal, BBTextField } from "@/bbkit";
import { payloadValueListToComponentList } from "@/components/SQLReview/components";
import { RichEngineName } from "@/components/v2";
import { SQLReviewRuleLevel } from "@/types/proto-es/v1/org_policy_service_pb";
import type { RuleTemplateV2 } from "@/types/sqlReview";
import { getRuleLocalization } from "@/types/sqlReview";
import RuleConfig from "./RuleConfigComponents/RuleConfig.vue";
import RuleLevelSwitch from "./RuleLevelSwitch.vue";

type LocalState = {
  level: SQLReviewRuleLevel;
  comment: string;
};

const props = defineProps<{
  rule: RuleTemplateV2;
  disabled: boolean;
}>();

const emit = defineEmits<{
  (event: "update:rule", update: Partial<RuleTemplateV2>): void;
  (event: "cancel"): void;
}>();

const ruleConfig = ref<InstanceType<typeof RuleConfig>>();

const state = reactive<LocalState>({
  level: props.rule.level,
  comment:
    props.rule.comment ||
    getRuleLocalization(props.rule.type, props.rule.engine).description,
});

const toggleActivity = (on: boolean) => {
  state.level = on ? SQLReviewRuleLevel.WARNING : SQLReviewRuleLevel.DISABLED;
};

watch(
  () => props.rule.level,
  () => {
    state.level = props.rule.level;
  }
);

const confirm = () => {
  emit("update:rule", {
    componentList: payloadValueListToComponentList(
      props.rule,
      ruleConfig.value?.payload ?? []
    ),
    level: state.level,
    comment: state.comment,
  });
  nextTick(() => {
    emit("cancel");
  });
};
</script>
