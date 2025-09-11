<template>
  <div class="sm:col-span-3 sm:col-start-1">
    <div class="flex items-center">
      <NRadioGroup v-model:value="state.mode">
        <NRadio value="ServiceName">
          <span class="textlabel">ServiceName</span>
        </NRadio>
        <NRadio value="SID">
          <span class="textlabel">SID</span>
        </NRadio>
      </NRadioGroup>
      <RequiredStar />
    </div>
    <div class="mt-2">
      <NInput
        :value="state.value"
        :placeholder="state.mode"
        @update:value="onUpdate"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput, NRadio, NRadioGroup } from "naive-ui";
import { reactive, watch } from "vue";
import RequiredStar from "@/components/RequiredStar.vue";

type Mode = "ServiceName" | "SID";

type LocalState = {
  mode: Mode;
  value: string;
};

const props = defineProps<{
  sid: string;
  serviceName: string;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (name: "update:sid", sid: string): void;
  (name: "update:serviceName", serviceName: string): void;
}>();

const guessModeFromProps = (): Mode => {
  if (props.sid) return "SID";
  return "ServiceName";
};

const state = reactive<LocalState>({
  value: props.serviceName || props.sid,
  mode: guessModeFromProps(),
});

const onUpdate = (value: string) => {
  if (state.mode === "SID") {
    emit("update:sid", value);
    emit("update:serviceName", "");
  } else {
    emit("update:serviceName", value);
    emit("update:sid", "");
  }
  state.value = value;
};

watch(
  () => state.mode,
  () => {
    onUpdate("");
  }
);
</script>
