<template>
  <div class="mt-2 sm:col-span-1 sm:col-start-1">
    <div class="flex items-center gap-x-4">
      <NRadio
        :checked="state.mode === 'ServiceName'"
        @update:checked="toggleMode('ServiceName', $event)"
      >
        <span class="textlabel">ServiceName</span>
      </NRadio>
      <NRadio
        :checked="state.mode === 'SID'"
        @update:checked="toggleMode('SID', $event)"
      >
        <span class="textlabel">SID</span>
      </NRadio>
    </div>
    <div class="mt-1">
      <NInput
        v-if="state.mode === 'ServiceName'"
        :value="state.serviceName"
        placeholder="ServiceName"
        @update:value="update($event, 'ServiceName')"
      />
      <NInput
        v-if="state.mode === 'SID'"
        :value="state.sid"
        placeholder="SID"
        @update:value="update($event, 'SID')"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NInput, NRadio } from "naive-ui";
import { reactive, watch } from "vue";

type Mode = "ServiceName" | "SID";

type LocalState = {
  mode: Mode;
  sid: string;
  serviceName: string;
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
  sid: props.sid,
  serviceName: props.serviceName,
  mode: guessModeFromProps(),
});

const update = (value: string, mode: Mode) => {
  if (mode === "SID") {
    emit("update:sid", value);
    emit("update:serviceName", "");
  } else {
    emit("update:serviceName", value);
    emit("update:sid", "");
  }
};

const toggleMode = (mode: Mode, checked: boolean) => {
  if (!checked) return;
  state.mode = mode;
};

watch(
  [() => props.sid, () => props.serviceName],
  ([sid, serviceName]) => {
    state.sid = sid;
    state.serviceName = serviceName;
  },
  { immediate: true }
);

watch(
  () => state.mode,
  (newMode, oldMode) => {
    // carry the input value when switching mode
    const oldValue = oldMode === "SID" ? state.sid : state.serviceName;
    update(oldValue, newMode);
  }
);
</script>
