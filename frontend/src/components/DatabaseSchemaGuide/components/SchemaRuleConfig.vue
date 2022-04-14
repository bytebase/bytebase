<template>
  <div>
    <div
      class="flex justify-center items-center py-4 px-2 group cursor-pointer hover:bg-gray-100"
      :class="active ? 'bg-gray-100' : ''"
      @click="$emit('activate', selectedRule.id)"
    >
      <heroicons-solid:chevron-right
        class="w-5 h-5 transform transition-all"
        :class="active ? 'rotate-90' : ''"
      />

      <div class="flex-1 flex flex-col ml-3">
        <div class="flex mb-2 items-center space-x-2">
          <h1 class="text-base font-semibold text-gray-900">
            {{ selectedRule.id }}
          </h1>
          <!-- <BBBadge :text="rule.category" :can-remove="false" /> -->
          <BBBadge
            v-for="db in selectedRule.database"
            :key="`${selectedRule.id}-${db}`"
            :text="db"
            :can-remove="false"
          />
          <SchemaRuleLevelBadge :level="selectedRule.level" />
        </div>
        <div class="text-sm text-gray-400">
          {{ selectedRule.description }}
        </div>
      </div>
      <div
        class="flex items-center p-2 mr-3 rounded opacity-0 cursor-pointer hover:bg-red-200 group-hover:opacity-100"
        @click="$emit('remove', selectedRule)"
      >
        <heroicons-outline:trash class="w-5 h-5 text-red-400" />
      </div>
    </div>
    <div v-if="active" class="px-10 py-5 text-sm">
      <div class="mb-7">
        <p class="mb-3">Level</p>
        <div class="flex gap-x-3">
          <div
            v-for="(level, index) in levelList"
            :key="index"
            class="flex items-center"
          >
            <input
              :id="`level-${level.id}`"
              :value="level.id"
              type="radio"
              :checked="level.id === selectedRule.level"
              @input="emit('level-change', level.id)"
              class="h-4 w-4 border-gray-300 rounded text-indigo-600 focus:ring-indigo-500"
            />
            <label
              :for="`level-${level.id}`"
              class="ml-2 items-center text-sm text-gray-600"
            >
              {{ level.name }}
            </label>
          </div>
        </div>
      </div>
      <div v-if="selectedRule.payload">
        <div
          v-for="[key, payload] in Object.entries(selectedRule.payload)"
          :key="key"
          class="mb-7"
        >
          <p class="mb-3">
            {{ `${key[0].toUpperCase()}${key.slice(1).toLowerCase()}` }}
          </p>
          <input
            v-if="payload.type == 'string'"
            v-model="state.payload[key]"
            type="text"
            class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md"
            :placeholder="payload.default"
          />
          <div v-else-if="payload.type == 'string[]'">
            <div class="flex flex-wrap gap-4 mb-4">
              <BBBadge
                v-for="(val, index) in state.payload[key]"
                :key="index"
                :text="val"
                @remove="() => removeFromList(key, val)"
              />
            </div>
            <input
              type="text"
              pattern="[a-z]+"
              class="shadow-sm focus:ring-indigo-500 focus:border-indigo-500 block w-full border-gray-300 rounded-md"
              placeholder="Input the value then press enter to add"
              @keyup.enter="(e) => pushToList(key, e)"
            />
          </div>
          <InputWithTemplate
            v-else-if="payload.type == 'template'"
            :templates="payload.templates"
            :value="state.payload[key]"
            @change="(val) => (state.payload[key] = val)"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { reactive, PropType, watch } from "vue";
import {
  levelList,
  SelectedRule,
  RulePayload,
} from "../../../types/schemaSystem";

interface LocalState {
  payload: {
    [val: string]: any;
  };
}

const initStatePayload = (
  payload: RulePayload | undefined
): { [val: string]: any } => {
  return Object.entries(payload ?? {}).reduce((res, [key, val]) => {
    res[key] = val.value ?? val.default;
    return res;
  }, {} as { [key: string]: any });
};

const props = defineProps({
  selectedRule: {
    required: true,
    type: Object as PropType<SelectedRule>,
  },
  active: {
    require: true,
    type: Boolean,
  },
});

const emit = defineEmits([
  "activate",
  "remove",
  "payload-change",
  "level-change",
]);

const state = reactive<LocalState>({
  payload: initStatePayload(props.selectedRule.payload),
});

watch(
  () => state.payload,
  (val) => emit("payload-change", val),
  { deep: true }
);

const removeFromList = (key: string, val: any) => {
  if (!Array.isArray(state.payload[key])) {
    return;
  }

  const values: Array<any> = state.payload[key];
  const index = values.indexOf(val);
  if (index < 0) {
    return;
  }

  state.payload[key] = [...values.slice(0, index), ...values.slice(index + 1)];
};

const pushToList = (key: string, e: any) => {
  if (!Array.isArray(state.payload[key])) {
    return;
  }

  const val = e.target.value.trim();
  state.payload[key].push(val);

  e.target.value = "";
};
</script>
