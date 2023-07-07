<template>
  <div class="grid grid-cols-2 gap-x-2 gap-y-1">
    <div>
      <label class="textlabel">
        {{ $t("instance.project-id") }}
        <span style="color: red">*</span>
      </label>
      <input
        v-model="state.projectId"
        required
        type="text"
        placeholder="projectId"
        class="textfield mt-1 w-full"
        :class="[
          state.dirty && !isValidProjectId && '!border-error !ring-error',
        ]"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <label class="textlabel">
        {{ $t("instance.instance-id") }}
        <span style="color: red">*</span>
      </label>
      <input
        v-model="state.instanceId"
        required
        type="text"
        placeholder="instanceId"
        class="textfield mt-1 w-full"
        :class="[
          state.dirty && !isValidInstanceId && '!border-error !ring-error',
        ]"
        :disabled="!allowEdit"
      />
    </div>

    <p class="col-span-2 textinfolabel">
      {{ $t("instance.find-gcp-project-id-and-instance-id") }}
      <a
        href="https://www.bytebase.com/docs/get-started/instance/#specify-google-cloud-project-id-and-spanner-instance-id"
        target="_blank"
        class="normal-link inline-flex items-center"
      >
        {{ $t("common.detailed-guide")
        }}<heroicons-outline:external-link class="w-4 h-4 ml-1"
      /></a>
    </p>
  </div>
</template>

<script lang="ts" setup>
import { computed, reactive, watch } from "vue";

type LocalState = {
  projectId: string;
  instanceId: string;
  dirty: boolean;
};

const props = defineProps<{
  host: string;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (name: "update:host", host: string): void;
}>();

const RE =
  /^projects\/(?<PROJECT_ID>(?:[a-z]|[-.:]|[0-9])*)\/instances\/(?<INSTANCE_ID>(?:[a-z]|[-]|[0-9])*)$/;
const RE_PROJECT_ID = /^(?:[a-z]|[-.:]|[0-9])+$/;
const RE_INSTANCE_ID = /^(?:[a-z]|[-]|[0-9])+$/;

const state = reactive<LocalState>({
  projectId: "",
  instanceId: "",
  dirty: false,
});

const isValidProjectId = computed(() => {
  return RE_PROJECT_ID.test(state.projectId);
});

const isValidInstanceId = computed(() => {
  return RE_INSTANCE_ID.test(state.instanceId);
});

const update = () => {
  if (!isValidProjectId.value || !isValidInstanceId.value) {
    emit("update:host", "");
    return;
  }
  const host = `projects/${state.projectId}/instances/${state.instanceId}`;
  emit("update:host", host);
};

const parseProjectIdFromHost = (host: string) => {
  const match = host.match(RE);
  return match?.groups?.PROJECT_ID ?? "";
};

const parseInstanceIdFromHost = (host: string) => {
  const match = host.match(RE);
  return match?.groups?.INSTANCE_ID ?? "";
};

watch([() => state.projectId, () => state.instanceId], () => {
  state.dirty = true;
  update();
});

watch(
  () => props.host,
  (host) => {
    if (!host) return;
    state.projectId = parseProjectIdFromHost(host);
    state.instanceId = parseInstanceIdFromHost(host);
  },
  { immediate: true }
);
</script>
