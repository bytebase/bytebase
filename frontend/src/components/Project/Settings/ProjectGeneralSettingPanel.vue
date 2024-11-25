<template>
  <form class="w-full space-y-4 mx-auto">
    <p class="text-lg font-medium leading-7 text-main">
      {{ $t("common.general") }}
    </p>
    <div class="flex justify-start items-start gap-6">
      <dl class="">
        <dt class="text-sm font-medium text-control-light">
          {{ $t("common.name") }} <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <NInput
            id="projectName"
            v-model:value="state.title"
            :disabled="!allowEdit"
            required
          />
        </dd>
        <div class="mt-1">
          <ResourceIdField
            resource-type="project"
            :value="extractProjectResourceName(project.name)"
            :readonly="true"
          />
        </div>
      </dl>

      <dl class="">
        <dt class="flex text-sm font-medium text-control-light">
          {{ $t("common.key") }}
          <NTooltip>
            <template #trigger>
              <heroicons-outline:information-circle class="ml-1 w-4 h-auto" />
            </template>
            {{ $t("project.key-hint") }}
          </NTooltip>
          <span class="text-red-600">*</span>
        </dt>
        <dd class="mt-1 text-sm text-main">
          <NInput
            id="projectKey"
            v-model:value="state.key"
            :disabled="!allowEdit"
            required
            @update:value="(val: string) => (state.key = val.toUpperCase())"
          />
        </dd>
      </dl>
    </div>

    <div v-if="allowEdit" class="flex justify-end">
      <NButton type="primary" :disabled="!allowSave" @click.prevent="save">
        {{ $t("common.update") }}
      </NButton>
    </div>

    <FeatureModal
      :open="!!state.requiredFeature"
      :feature="state.requiredFeature"
      @cancel="state.requiredFeature = undefined"
    />
  </form>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty } from "lodash-es";
import { NButton, NInput, NTooltip } from "naive-ui";
import { computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { FeatureModal } from "@/components/FeatureGuard";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { pushNotification, useProjectV1Store } from "@/store";
import type { FeatureType } from "@/types";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Project } from "@/types/proto/v1/project_service";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  title: string;
  key: string;
  requiredFeature: FeatureType | undefined;
}

const props = defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  title: props.project.title,
  key: props.project.key,
  requiredFeature: undefined,
});

const allowSave = computed((): boolean => {
  return (
    props.project.name !== DEFAULT_PROJECT_NAME &&
    !isEmpty(state.title) &&
    (state.title !== props.project.title || state.key !== props.project.key)
  );
});

const save = () => {
  const projectPatch = cloneDeep(props.project);
  const updateMask: string[] = [];
  if (state.title !== props.project.title) {
    projectPatch.title = state.title;
    updateMask.push("title");
  }
  if (state.key !== props.project.key) {
    projectPatch.key = state.key;
    updateMask.push("key");
  }
  projectV1Store.updateProject(projectPatch, updateMask).then(() => {
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.settings.success-updated"),
    });
  });
};
</script>
