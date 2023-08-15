<template>
  <BBModal :title="$t('sql-review.edit-rule.self')" @close="$emit('cancel')">
    <div class="space-y-4 w-[calc(100vw-5rem)] sm:w-[40rem] pb-1">
      <div class="space-y-1">
        <h3 class="text-lg text-control font-medium">
          {{ $t("common.name") }}
        </h3>
        <div class="textinfolabel flex items-center gap-x-2">
          {{ getRuleLocalization(rule.type).title }}
          <RuleEngineIcons :rule="rule" />
          <a
            :href="`https://www.bytebase.com/docs/sql-review/review-rules#${rule.type}`"
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
          <BBSwitch
            :class="[!editable && 'pointer-events-none']"
            :disabled="disabled"
            :value="state.level !== RuleLevel.DISABLED"
            size="small"
            @toggle="toggleActivity(rule, $event)"
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
            :editable="editable"
            @level-change="state.level = $event"
          />
        </div>
      </div>
      <div v-if="editable" class="space-y-1">
        <h3 class="text-lg text-control font-medium">
          {{ $t("common.description") }}
        </h3>
        <div class="flex flex-col gap-x-2">
          <AutoHeightTextarea
            v-model:value="state.comment"
            :disabled="disabled"
            :placeholder="
              getRuleLocalization(rule.type).description ||
              $t('common.description')
            "
            rows="1"
            :max-height="120"
          />
        </div>
      </div>
      <div v-else-if="displayDescription" class="space-y-1">
        <h3 class="text-lg text-control font-medium">
          {{ $t("common.description") }}
        </h3>
        <div class="flex flex-col gap-x-2">
          {{ displayDescription }}
        </div>
      </div>
      <RuleEngineTabFilter
        v-if="rule.individualConfigList.length > 0"
        :selected="state.selectedEngine"
        :engine-list="rule.engineList"
        :individual-engine-list="rule.individualConfigList.map((c) => c.engine)"
        @update:engine="(val: string) => state.selectedEngine = val"
      />
      <div
        v-for="(config, index) in rule.componentList"
        :key="index"
        class="space-y-1"
      >
        <p class="text-lg text-control font-medium mb-2">
          {{ configTitle(config) }}
        </p>
        <StringComponent
          v-if="config.payload.type === 'STRING'"
          :value="state.payload[state.selectedEngine][index] as string"
          :config="config"
          :disabled="disabled"
          :editable="editable"
          @update:value="state.payload[state.selectedEngine][index] = $event"
        />
        <NumberComponent
          v-if="config.payload.type === 'NUMBER'"
          :value="state.payload[state.selectedEngine][index] as number"
          :config="config"
          :disabled="disabled"
          :editable="editable"
          @update:value="state.payload[state.selectedEngine][index] = $event"
        />
        <BooleanComponent
          v-else-if="config.payload.type == 'BOOLEAN'"
          :rule="rule"
          :value="state.payload[state.selectedEngine][index] as boolean"
          :config="config"
          :disabled="disabled"
          :editable="editable"
          @update:value="state.payload[state.selectedEngine][index] = $event"
        />
        <StringArrayComponent
          v-else-if="
            config.payload.type == 'STRING_ARRAY' &&
            Array.isArray(state.payload[state.selectedEngine][index])
          "
          :value="state.payload[state.selectedEngine][index] as string[]"
          :config="config"
          :disabled="disabled"
          :editable="editable"
          @update:value="state.payload[state.selectedEngine][index] = $event"
        />
        <TemplateComponent
          v-else-if="config.payload.type == 'TEMPLATE'"
          :rule="rule"
          :value="state.payload[state.selectedEngine][index] as string"
          :config="config"
          :disabled="disabled"
          :editable="editable"
          @update:value="state.payload[state.selectedEngine][index] = $event"
        />
      </div>
      <div v-if="editable" class="mt-4 pt-2 border-t flex justify-end">
        <button
          type="button"
          class="btn-normal py-2 px-4"
          @click.prevent="$emit('cancel')"
        >
          {{ $t("common.cancel") }}
        </button>
        <button
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          :disabled="disabled"
          @click.prevent="confirm"
        >
          {{ $t("common.confirm") }}
        </button>
      </div>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { computed, nextTick, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import AutoHeightTextarea from "@/components/misc/AutoHeightTextarea.vue";
import { UNKNOWN_ID } from "@/types";
import {
  getRuleLocalization,
  getRuleLocalizationKey,
  RuleConfigComponent,
  RuleLevel,
  RuleTemplate,
} from "@/types/sqlReview";
import {
  StringComponent,
  NumberComponent,
  BooleanComponent,
  StringArrayComponent,
  TemplateComponent,
  PayloadValueType,
  PayloadForEngine,
} from "./RuleConfigComponents";
import RuleEngineIcons from "./RuleEngineIcons.vue";
import RuleLevelSwitch from "./RuleLevelSwitch.vue";

type LocalState = {
  payload: PayloadForEngine;
  level: RuleLevel;
  comment: string;
  selectedEngine: string;
};

const props = defineProps<{
  editable: boolean;
  rule: RuleTemplate;
  disabled: boolean;
}>();

const emit = defineEmits<{
  (event: "update:payload", payload: PayloadForEngine): void;
  (event: "update:level", level: RuleLevel): void;
  (event: "update:comment", comment: string): void;
  (event: "cancel"): void;
}>();

const { t } = useI18n();

const getRulePayload = () => {
  const { componentList, individualConfigList, engineList } = props.rule;
  const resp: PayloadForEngine = {};

  if (componentList.length === 0) {
    return resp;
  }

  const basePayload = componentList.reduce<
    { key: string; value: PayloadValueType }[]
  >((list, component) => {
    list.push({
      key: component.key,
      value: component.payload.value ?? component.payload.default,
    });
    return list;
  }, []);

  if (engineList.length > individualConfigList.length) {
    resp[`${UNKNOWN_ID}`] = basePayload.map((val) => val.value);
  }

  for (const individualConfig of individualConfigList) {
    const individualPayload = [...basePayload];
    for (const key of Object.keys(individualConfig.payload)) {
      const index = individualPayload.findIndex((val) => val.key === key);
      if (index >= 0) {
        individualPayload[index] = {
          ...individualPayload[index],
          value:
            individualConfig.payload[key].value ??
            individualConfig.payload[key].default,
        };
      }
    }
    resp[individualConfig.engine] = individualPayload.map((val) => val.value);
  }

  return resp;
};

const state = reactive<LocalState>({
  payload: getRulePayload(),
  level: props.rule.level,
  comment:
    props.rule.comment || getRuleLocalization(props.rule.type).description,
  selectedEngine: `${UNKNOWN_ID}`,
});

const displayDescription = computed(() => {
  return state.comment || getRuleLocalization(props.rule.type).description;
});

const configTitle = (config: RuleConfigComponent): string => {
  const key = `sql-review.rule.${getRuleLocalizationKey(
    props.rule.type
  )}.component.${config.key}.title`;
  return t(key);
};

const toggleActivity = (rule: RuleTemplate, on: boolean) => {
  state.level = on ? RuleLevel.WARNING : RuleLevel.DISABLED;
};

watch(
  () => props.rule.level,
  () => {
    state.level = props.rule.level;
  }
);

const confirm = () => {
  emit("update:level", state.level);
  emit("update:payload", state.payload);
  emit("update:comment", state.comment);
  nextTick(() => {
    emit("cancel");
  });
};
</script>
