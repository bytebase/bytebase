<template>
  <div class="grid grid-cols-2 gap-x-2 gap-y-1">
    <div>
      <label class="textlabel">
        {{ $t("instance.project-id") }}
        <span style="color: red">*</span>
      </label>
      <input
        v-model="projectId"
        required
        type="text"
        placeholder="projectId"
        class="textfield mt-1 w-full"
        :disabled="!allowEdit"
      />
    </div>
    <div>
      <label class="textlabel">
        {{ $t("instance.instance-id") }}
        <span style="color: red">*</span>
      </label>
      <input
        v-model="instanceId"
        required
        type="text"
        placeholder="instanceId"
        class="textfield mt-1 w-full"
        :disabled="!allowEdit"
      />
    </div>

    <p class="col-span-2 textinfolabel">
      {{ $t("instance.find-gcp-project-id-and-instance-id") }}
      <a
        href="https://www.bytebase.com/docs/how-to/spanner/how-to-find-project-id-and-instance-id"
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

const props = defineProps<{
  host: string;
  allowEdit: boolean;
}>();

const emit = defineEmits<{
  (name: "update:host", host: string): void;
}>();

const RE =
  /^projects\/(?<PROJECT_ID>(?:[a-z]|[-.:]|[0-9])*)\/instances\/(?<INSTANCE_ID>(?:[a-z]|[-]|[0-9])*)/;

const state = reactive({
  host: props.host,
});

const update = (projectId: string, instanceId: string) => {
  const host = `projects/${projectId}/instances/${instanceId}`;
  emit("update:host", host);
};

const projectId = computed<string>({
  get() {
    const match = state.host.match(RE);
    return match?.groups?.PROJECT_ID ?? "";
  },
  set(value) {
    update(value, instanceId.value);
  },
});

const instanceId = computed<string>({
  get() {
    const match = state.host.match(RE);
    return match?.groups?.INSTANCE_ID ?? "";
  },
  set(value) {
    update(projectId.value, value);
  },
});

watch(
  () => props.host,
  (host) => (state.host = host)
);
</script>
