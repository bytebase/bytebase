<template>
  <GrantAccessForm
    :title="$t('project.masking-exemption.grant-exemption')"
    :column-list="[]"
    :project-name="project.name"
    @dismiss="() => router.back()"
  />
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import GrantAccessForm from "@/components/SensitiveData/GrantAccessForm.vue";
import { useBodyLayoutContext } from "@/layouts/common";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const router = useRouter();

const { overrideMainContainerClass } = useBodyLayoutContext();

overrideMainContainerClass("!pb-0");
</script>
