<template>
  <form class="w-full space-y-4">
    <div>
      <div class="font-medium">
        {{ $t("common.name") }}
        <RequiredStar />
      </div>
      <NInput
        id="projectName"
        class="mt-1"
        v-model:value="state.title"
        :disabled="!allowEdit"
        required
      />
      <div class="mt-1">
        <ResourceIdField
          resource-type="project"
          :value="extractProjectResourceName(project.name)"
          :readonly="true"
        />
      </div>
    </div>
  </form>
</template>

<script lang="ts" setup>
import { cloneDeep, isEmpty } from "lodash-es";
import { NInput } from "naive-ui";
import { computed, reactive } from "vue";
import RequiredStar from "@/components/RequiredStar.vue";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { useProjectV1Store } from "@/store";
import { DEFAULT_PROJECT_NAME } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { extractProjectResourceName } from "@/utils";

interface LocalState {
  title: string;
}

const props = defineProps<{
  project: Project;
  allowEdit: boolean;
}>();

const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  title: props.project.title,
});

const allowSave = computed((): boolean => {
  return (
    props.project.name !== DEFAULT_PROJECT_NAME &&
    !isEmpty(state.title) &&
    state.title !== props.project.title
  );
});

const onUpdate = async () => {
  const projectPatch = cloneDeep(props.project);
  if (state.title !== props.project.title) {
    projectPatch.title = state.title;
    await projectV1Store.updateProject(projectPatch, ["title"]);
  }
};

defineExpose({
  isDirty: allowSave,
  update: onUpdate,
  revert: () => {
    state.title = props.project.title;
  },
});
</script>
