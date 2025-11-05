<template>
  <Drawer :show="true" @close="() => emit('dismiss')">
    <DrawerContent
      :title="$t('settings.sensitive-data.grant-access')"
      class="w-5xl max-w-[100vw] relative grant-access-drawer"
    >
      <GrantAccessForm
        :column-list="columnList"
        :project-name="projectName"
        @dismiss="() => emit('dismiss')"
      />
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import GrantAccessForm from "@/components/SensitiveData/GrantAccessForm.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import type { SensitiveColumn } from "./types";

defineProps<{
  columnList: SensitiveColumn[];
  projectName: string;
}>();

const emit = defineEmits(["dismiss"]);
</script>

<style scoped lang="postcss">
.grant-access-drawer :deep(.n-drawer-body-content-wrapper) {
  padding-bottom: 0 !important;
}
</style>
