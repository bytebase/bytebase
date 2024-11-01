<template>
  <div class="h-full flex flex-col">
    <div class="text-lg leading-6 font-medium text-main">
      {{ $t("project.masking-access.grant-access") }}
    </div>
    <NDivider />
    <GrantAccessForm
      class="flex-1"
      :column-list="[]"
      :project-name="project.name"
      @dismiss="() => router.back()"
    />
  </div>
</template>

<script lang="tsx" setup>
import { NDivider } from "naive-ui";
import { computed } from "vue";
import { useRouter } from "vue-router";
import GrantAccessForm from "@/components/SensitiveData/GrantAccessForm.vue";
import { useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";

const props = defineProps<{
  projectId: string;
}>();

const { project } = useProjectByName(
  computed(() => `${projectNamePrefix}${props.projectId}`)
);
const router = useRouter();
</script>
