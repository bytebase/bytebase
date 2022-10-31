<template>
  <div :id="selectedRule.type.replace(/\./g, '-')">
    <div
      class="flex justify-center items-center py-4 px-2 group cursor-pointer hover:bg-gray-100"
      :class="active ? 'bg-gray-100' : ''"
      @click="$emit('activate', selectedRule.type)"
    >
      <div class="flex-1 flex flex-col">
        <div class="flex items-center">
          <div class="flex flex-1 items-center space-x-2">
            <h1 class="flex text-base gap-x-1">
              <NTooltip v-if="disabled" trigger="hover" :show-arrow="false">
                <template #trigger>
                  <div class="flex justify-center">
                    <heroicons-outline:exclamation
                      class="h-6 w-6 text-yellow-600"
                    />
                  </div>
                </template>
                <span class="whitespace-nowrap">
                  {{
                    $t("sql-review.not-available-for-free", {
                      plan: $t(
                        `subscription.plan.${planTypeToString(
                          subscriptionStore.currentPlan
                        )}.title`
                      ),
                    })
                  }}
                </span>
              </NTooltip>
              {{ getRuleLocalization(selectedRule.type).title }}
            </h1>
            <SQLRuleLevelBadge :level="selectedRule.level" />
            <img
              v-for="engine in selectedRule.engineList"
              :key="engine"
              class="h-4 w-auto"
              :src="getEngineIcon(engine)"
            />
            <a
              :href="`https://www.bytebase.com/docs/sql-review/review-rules/supported-rules#${selectedRule.type}`"
              target="__blank"
              class="flex flex-row space-x-2 items-center text-base text-gray-500 hover:text-gray-900"
            >
              <heroicons-outline:external-link class="w-4 h-4" />
            </a>
          </div>
          <heroicons-solid:chevron-right
            class="w-5 h-5 transform transition-all order-last"
            :class="active ? 'rotate-90' : ''"
          />
        </div>
        <div class="text-sm text-gray-400">
          {{ getRuleLocalization(selectedRule.type).description }}
        </div>
      </div>
    </div>
    <div v-if="active" class="px-5 py-5 text-sm">
      <div class="mb-7">
        <p class="mb-3">{{ $t("sql-review.level.name") }}</p>
        <div class="flex gap-x-3">
          <div
            v-for="(level, index) in LEVEL_LIST"
            :key="index"
            class="flex items-center"
          >
            <input
              :id="`level-${level}`"
              :value="level"
              type="radio"
              :disabled="disabled"
              :checked="level === selectedRule.level"
              :class="[
                'text-accent disabled:text-accent-disabled focus:ring-accent',
                disabled ? 'cursor-not-allowed' : '',
              ]"
              @input="emit('level-change', level)"
            />
            <label
              :for="`level-${level}`"
              :class="[
                'ml-2 items-center text-sm text-gray-600',
                disabled ? 'cursor-not-allowed' : '',
              ]"
            >
              {{ $t(`sql-review.level.${level.toLowerCase()}`) }}
            </label>
          </div>
        </div>
      </div>
      <div
        v-for="(config, index) in selectedRule.componentList"
        :key="index"
        class="mb-4"
      >
        <p class="mb-3">
          {{
            $t(
              `sql-review.rule.${getRuleLocalizationKey(
                selectedRule.type
              )}.component.${config.key}.title`
            )
          }}
        </p>
        <input
          v-if="config.payload.type == 'STRING'"
          v-model="state.payload[index]"
          type="text"
          :disabled="disabled"
          :class="[
            'shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md',
            disabled ? 'cursor-not-allowed' : '',
          ]"
          :placeholder="config.payload.default"
        />
        <input
          v-else-if="config.payload.type == 'NUMBER'"
          v-model="state.payload[index]"
          type="number"
          :disabled="disabled"
          :class="[
            'shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md',
            disabled ? 'cursor-not-allowed' : '',
          ]"
          :placeholder="`${config.payload.default}`"
        />
        <BBCheckbox
          v-else-if="config.payload.type == 'BOOLEAN'"
          :title="
            $t(
              `sql-review.rule.${getRuleLocalizationKey(
                selectedRule.type
              )}.component.${config.key}.title`
            )
          "
          :value="state.payload[index]"
          @toggle="(on: boolean) => {
            state.payload[index] = on;
          }"
        />
        <div
          v-else-if="
            config.payload.type == 'STRING_ARRAY' &&
            Array.isArray(state.payload[index])
          "
        >
          <div class="flex flex-wrap gap-4 mb-4">
            <BBBadge
              v-for="(val, i) in state.payload[index]"
              :key="`${index}-${i}`"
              :text="`${val}`"
              :can-remove="!disabled"
              @remove="() => removeFromList(index, val)"
            />
          </div>
          <input
            type="text"
            pattern="[a-z]+"
            :disabled="disabled"
            :class="[
              'shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md',
              disabled ? 'cursor-not-allowed' : '',
            ]"
            :placeholder="$t('sql-review.input-then-press-enter')"
            @keyup.enter="(e) => pushToList(index, e)"
          />
        </div>
        <InputWithTemplate
          v-else-if="config.payload.type == 'TEMPLATE'"
          :template-list="
            config.payload.templateList.map((id) => ({
              id,
              description: $t(
                `sql-review.rule.${getRuleLocalizationKey(
                  selectedRule.type
                )}.component.${config.key}.template.${id}`
              ),
            }))
          "
          :value="getStringPayload(index)"
          @change="(val) => (state.payload[index] = val)"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, PropType, watch } from "vue";
import { pullAt } from "lodash-es";
import { NTooltip } from "naive-ui";
import {
  LEVEL_LIST,
  RuleTemplate,
  RuleConfigComponent,
  getRuleLocalization,
  getRuleLocalizationKey,
  SchemaRuleEngineType,
} from "@/types/sqlReview";
import { useSubscriptionStore } from "@/store";
import { planTypeToString } from "@/types/plan";

type PayloadValueList = (boolean | string | number | string[])[];
interface LocalState {
  payload: PayloadValueList;
}

const initStatePayload = (
  componentList: RuleConfigComponent[] | undefined
): PayloadValueList => {
  return (componentList ?? []).reduce((res, component) => {
    res.push(component.payload.value ?? component.payload.default);
    return res;
  }, [] as PayloadValueList);
};

const props = defineProps({
  selectedRule: {
    required: true,
    type: Object as PropType<RuleTemplate>,
  },
  active: {
    require: true,
    type: Boolean,
  },
  disabled: {
    require: false,
    default: false,
    type: Boolean,
  },
});

const emit = defineEmits(["activate", "payload-change", "level-change"]);

const state = reactive<LocalState>({
  payload: initStatePayload(props.selectedRule.componentList),
});

const subscriptionStore = useSubscriptionStore();

watch(
  () => state.payload,
  (val) => emit("payload-change", val),
  { deep: true }
);

const removeFromList = (i: number, val: any) => {
  if (!Array.isArray(state.payload[i])) {
    return;
  }

  const values = state.payload[i] as string[];
  const index = values.indexOf(val);
  pullAt(state.payload[i] as string[], index);
};

const pushToList = (i: number, e: any) => {
  if (!Array.isArray(state.payload[i])) {
    return;
  }

  const val = e.target.value.trim();
  if (val) {
    const existed = state.payload[i] as string[];
    if (!new Set(existed).has(val)) {
      existed.push(val);
      e.target.value = "";
    }
  }
};

const getStringPayload = (i: number): string => {
  return state.payload[i] as string;
};

const getEngineIcon = (engine: SchemaRuleEngineType) =>
  new URL(`../../../assets/db-${engine.toLowerCase()}.png`, import.meta.url)
    .href;
</script>

<style scoped>
/*  Removed the ticker in the number field  */
input::-webkit-outer-spin-button,
input::-webkit-inner-spin-button {
  -webkit-appearance: none;
  margin: 0;
}

/* Firefox */
input[type="number"] {
  -moz-appearance: textfield;
}
</style>
