<template>
  <div class="mt-2 sm:col-span-1 sm:col-start-1">
    <div class="flex items-center gap-x-4">
      <label class="radio">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="ServiceName"
          :disabled="!allowEdit"
        />
        <span class="label">ServiceName</span>
      </label>
      <label class="radio">
        <input
          v-model="state.mode"
          tabindex="-1"
          type="radio"
          class="text-accent disabled:text-accent-disabled focus:ring-accent"
          value="SID"
          :disabled="!allowEdit"
        />
        <span class="label">SID</span>
      </label>
    </div>
    <div>
      <input
        v-if="state.mode === 'ServiceName'"
        :value="state.serviceName"
        type="text"
        class="textfield w-full mt-1"
        placeholder="ServiceName"
        @input="
          update(($event.target as HTMLInputElement).value, 'ServiceName')
        "
      />
      <input
        v-if="state.mode === 'SID'"
        :value="state.sid"
        type="text"
        class="textfield w-full mt-1"
        placeholder="SID"
        @input="update(($event.target as HTMLInputElement).value, 'SID')"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
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
