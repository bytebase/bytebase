<template>
  <div>
    <div
      class="flex justify-center items-center py-4 px-2 group cursor-pointer hover:bg-gray-100"
      :class="active ? 'bg-gray-100' : ''"
      @click="$emit('activate', selectedRule.type)"
    >
      <div class="flex-1 flex flex-col">
        <div class="flex mb-2 items-center space-x-2">
          <heroicons-solid:chevron-right
            class="w-5 h-5 transform transition-all"
            :class="active ? 'rotate-90' : ''"
          />
          <h1 class="text-base font-semibold text-gray-900">
            {{ getRuleLocalization(selectedRule.type).title }}
          </h1>
          <BBBadge
            :text="$t(`engine.${selectedRule.engine.toLowerCase()}`)"
            :can-remove="false"
          />
          <SchemaRuleLevelBadge :level="selectedRule.level" />
        </div>
        <div class="text-sm text-gray-400 ml-7">
          {{ getRuleLocalization(selectedRule.type).description }}
        </div>
      </div>
    </div>
    <div v-if="active" class="px-10 py-5 text-sm">
      <div class="mb-7">
        <p class="mb-3">{{ $t("schema-review-policy.error-level.name") }}</p>
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
              :checked="level === selectedRule.level"
              @input="emit('level-change', level)"
              class="text-accent disabled:text-accent-disabled focus:ring-accent"
            />
            <label
              :for="`level-${level}`"
              class="ml-2 items-center text-sm text-gray-600"
            >
              {{
                $t(`schema-review-policy.error-level.${level.toLowerCase()}`)
              }}
            </label>
          </div>
        </div>
      </div>
      <div
        v-for="(config, index) in selectedRule.componentList"
        :key="index"
        class="mb-1"
      >
        <p class="mb-3">
          {{ $t(`schema-review-policy.payload-config.${config.title}`) }}
        </p>
        <input
          v-if="config.payload.type == 'STRING'"
          v-model="state.payload[index]"
          type="text"
          class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md"
          :placeholder="config.payload.default"
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
              :text="val"
              @remove="() => removeFromList(index, val)"
            />
          </div>
          <input
            type="text"
            pattern="[a-z]+"
            class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md"
            :placeholder="
              $t('schema-review-policy.payload-config.input-then-press-enter')
            "
            @keyup.enter="(e) => pushToList(index, e)"
          />
        </div>
        <InputWithTemplate
          v-else-if="config.payload.type == 'TEMPLATE'"
          :template-list="config.payload.templateList"
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
import {
  LEVEL_LIST,
  RuleTemplate,
  RuleConfigComponent,
  getRuleLocalization,
} from "@/types/schemaSystem";

type PayloadValueList = (string | string[])[];
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
});

const emit = defineEmits(["activate", "payload-change", "level-change"]);

const state = reactive<LocalState>({
  payload: initStatePayload(props.selectedRule.componentList),
});

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
  pullAt(state.payload[i], index);
};

const pushToList = (i: number, e: any) => {
  if (!Array.isArray(state.payload[i])) {
    return;
  }

  const val = e.target.value.trim();
  (state.payload[i] as string[]).push(val);

  e.target.value = "";
};

const getStringPayload = (i: number): string => {
  return state.payload[i] as string;
};
</script>
