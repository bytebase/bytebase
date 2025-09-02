<template>
  <div class="grid grid-cols-2 gap-x-2 gap-y-1">
    <div>
      <label class="textlabel">
        {{ $t("instance.project-id") }}
        <span style="color: red">*</span>
      </label>
      <NInput
        v-model:value="state.projectId"
        required
        placeholder="projectId"
        class="mt-1 w-full"
        :status="state.dirty && !isValidProjectId ? 'error' : undefined"
        :disabled="!allowEdit"
      />
    </div>

    <p class="col-span-2 textinfolabel">
      {{ $t("instance.find-gcp-project-id") }}
      <a
        href="https://docs.bytebase.com/get-started/connect/gcp?source=console"
        target="_blank"
        class="normal-link inline-flex items-center"
      >
        {{ $t("common.detailed-guide") }}
        <heroicons-outline:external-link class="w-4 h-4 ml-1" />
      </a>
    </p>
  </div>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { computed, reactive, watch } from "vue";

type LocalState = {
  projectId: string;
  dirty: boolean;
};

const props = defineProps<{
  host: string;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (name: "update:host", host: string): void;
}>();

const RE_PROJECT_ID = /^(?:[a-z]|[-.:]|[0-9])+$/;

const state = reactive<LocalState>({
  projectId: "",
  dirty: false,
});

const isValidProjectId = computed(() => {
  return RE_PROJECT_ID.test(state.projectId);
});

const update = () => {
  if (!isValidProjectId.value) {
    emit("update:host", "");
    return;
  }
  const host = `${state.projectId}`;
  emit("update:host", host);
};

watch([() => state.projectId], () => {
  state.dirty = true;
  update();
});

watch(
  () => props.host,
  (host) => {
    if (!host) return;
    state.projectId = host;
  },
  { immediate: true }
);
</script>
