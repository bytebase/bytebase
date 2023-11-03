<template>
  <div
    class="grid gap-y-4 gap-x-4 items-center text-sm"
    style="grid-template-columns: auto 1fr"
  >
    <div
      v-for="param in SupportedGhostParameters"
      :key="param.key"
      class="contents"
    >
      <div class="font-medium text-control">{{ param.key }}</div>
      <div class="textinfolabel break-all">
        <NInput
          v-if="param.type === 'int'"
          size="small"
          :value="getIntValue(param.key)"
          :allow-input="onlyAllowNumber"
          :disabled="readonly"
          @update:value="setIntValue(param.key, parseInt($event, 10))"
        />
        <BoolFlag
          v-if="param.type === 'bool'"
          :value="getBoolValue(param.key)"
          :defaults="getBoolValueDefaults(param.key)"
          :disabled="readonly"
          @update:value="setBoolValue(param.key, $event)"
        />
        <NInput
          v-if="param.type === 'string'"
          size="small"
          :value="getStringValue(param.key)"
          :disabled="readonly"
          @update:value="setStringValue(param.key, $event)"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { onlyAllowNumber } from "@/utils";
import BoolFlag from "./BoolFlag.vue";
import { DefaultGhostParameters, SupportedGhostParameters } from "./constants";

const props = defineProps<{
  flags: Record<string, string>;
  readonly: boolean;
}>();
const emit = defineEmits<{
  (event: "update:flags", flags: Record<string, string>): void;
}>();

const getBoolValue = (key: string) => {
  const value = props.flags[key];
  if (value === "true") return "true";
  if (value === "false") return "false";
  return "";
};
const getBoolValueDefaults = (key: string) => {
  const defaults = DefaultGhostParameters.find((p) => p.key === key);
  return defaults?.value === "true" ?? false;
};
const getStringValue = (key: string) => {
  return props.flags[key] ?? "";
};
const getIntValue = (key: string) => {
  const intVal = parseInt(props.flags[key], 10);
  if (Number.isNaN(intVal)) return undefined;
  return String(intVal);
};

const setBoolValue = (key: string, value: "true" | "false" | "") => {
  const updated = { ...props.flags };
  if (value !== "") {
    updated[key] = value;
  } else {
    delete updated[key];
  }
  emit("update:flags", updated);
};
const setStringValue = (key: string, value: string) => {
  const updated = { ...props.flags };
  value = value.trim();
  if (value) {
    updated[key] = value;
  } else {
    delete updated[key];
  }
  emit("update:flags", updated);
};
const setIntValue = (key: string, value: number) => {
  const updated = { ...props.flags };

  if (!Number.isNaN(value)) {
    updated[key] = value.toString();
  } else {
    delete updated[key];
  }
  emit("update:flags", updated);
};
</script>
