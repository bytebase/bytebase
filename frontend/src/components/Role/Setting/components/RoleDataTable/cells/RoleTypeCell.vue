<template>
  <NTooltip>
    <template #trigger>
      <component :is="roleTypeIcon" class="w-5 h-auto text-gray-500" />
    </template>
    {{ roleTypeTooltip }}
  </NTooltip>
</template>

<script lang="ts" setup>
import { BuildingIcon, GalleryHorizontalEndIcon } from "lucide-vue-next";
import { NTooltip } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { Role } from "@/types/proto/v1/role_service";
import { isProjectLevelRole, isWorkspaceLevelRole } from "@/utils";

const props = defineProps<{
  role: Role;
}>();

const { t } = useI18n();

const roleTypeIcon = computed(() => {
  if (isWorkspaceLevelRole(props.role.name)) {
    return BuildingIcon;
  } else if (isProjectLevelRole(props.role.name)) {
    return GalleryHorizontalEndIcon;
  } else {
    // Should never reach here.
    return null;
  }
});

const roleTypeTooltip = computed(() => {
  if (isWorkspaceLevelRole(props.role.name)) {
    return t("common.workspace");
  } else if (isProjectLevelRole(props.role.name)) {
    return t("common.project");
  } else {
    // Should never reach here.
    return null;
  }
});
</script>
