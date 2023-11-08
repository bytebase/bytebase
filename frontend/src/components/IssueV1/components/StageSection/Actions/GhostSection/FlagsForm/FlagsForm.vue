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
          :value="getIntValue(param)"
          :allow-input="onlyAllowNumber"
          :disabled="readonly"
          :placeholder="param.defaults"
          @update:value="setIntValue(param, parseInt($event, 10))"
        />
        <NInput
          v-if="param.type === 'float'"
          size="small"
          :value="getFloatValue(param)"
          :disabled="readonly"
          :placeholder="param.defaults"
          @update:value="setFloatValue(param, $event)"
        />
        <BoolFlag
          v-if="param.type === 'bool'"
          :value="getBoolValue(param)"
          :defaults="param.defaults === 'true'"
          :readonly="readonly"
          @update:value="setBoolValue(param, $event)"
        />
        <NInput
          v-if="param.type === 'string'"
          size="small"
          :value="getStringValue(param)"
          :disabled="readonly"
          :placeholder="param.defaults"
          @update:value="setStringValue(param, $event)"
        />
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { onlyAllowNumber } from "@/utils";
import BoolFlag from "./BoolFlag.vue";
import { GhostParameter, SupportedGhostParameters } from "./constants";

const props = defineProps<{
  flags: Record<string, string>;
  readonly: boolean;
}>();
const emit = defineEmits<{
  (event: "update:flags", flags: Record<string, string>): void;
}>();

const getBoolValue = (param: GhostParameter<"bool">) => {
  const { key, defaults: value } = param;
  if (props.flags[key] === "true") return true;
  if (props.flags[key] === "false") return false;
  return value === "true";
};
const getStringValue = (param: GhostParameter<"string">) => {
  return props.flags[param.key] ?? "";
};
const getIntValue = (param: GhostParameter<"int">) => {
  const intVal = parseInt(props.flags[param.key], 10);
  if (Number.isNaN(intVal)) return undefined;
  return String(intVal);
};
const getFloatValue = (param: GhostParameter<"float">) => {
  return props.flags[param.key];
};

const setBoolValue = (param: GhostParameter, value: boolean) => {
  const { key } = param;
  const updated = { ...props.flags };
  updated[key] = value ? "true" : "false";
  if (updated[key] === param.defaults) {
    delete updated[key];
  }
  emit("update:flags", updated);
};
const setStringValue = (param: GhostParameter<"string">, value: string) => {
  const updated = { ...props.flags };
  const { key } = param;
  value = value.trim();
  if (value) {
    updated[key] = value;
  } else {
    delete updated[key];
  }
  emit("update:flags", updated);
};
const setIntValue = (param: GhostParameter<"int">, value: number) => {
  const updated = { ...props.flags };
  const { key } = param;
  if (!Number.isNaN(value)) {
    updated[key] = value.toString();
  } else {
    delete updated[key];
  }
  emit("update:flags", updated);
};
const setFloatValue = (param: GhostParameter<"float">, value: string) => {
  const updated = { ...props.flags };
  const { key } = param;
  value = value.trim();
  if (value !== "") {
    updated[key] = value;
  } else {
    delete updated[key];
  }
  emit("update:flags", updated);
};
</script>
