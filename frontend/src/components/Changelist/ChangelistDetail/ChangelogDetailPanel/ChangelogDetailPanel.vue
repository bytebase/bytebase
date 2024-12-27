<template>
  <Drawer :show="show" @close="$emit('close')">
    <DrawerContent
      style="width: 75vw; max-width: calc(100vw - 8rem)"
      :title="$t('common.change-history')"
    >
      <ChangelogDetail v-if="detailBindings" v-bind="detailBindings" />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useChangelogStore } from "@/store";
import { extractDatabaseResourceName } from "@/utils";
import { extractChangelogUID } from "@/utils/v1/changelog";
import ChangelogDetail from "@/views/DatabaseDetail/ChangelogDetail.vue";
import { provideChangelistDetailContext } from "../context";

const props = defineProps<{
  changelogName?: string;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { project } = provideChangelistDetailContext();

const changelog = computed(() => {
  const { changelogName } = props;
  if (!changelogName) {
    return undefined;
  }
  return useChangelogStore().getChangelogByName(changelogName);
});

const detailBindings = computed(() => {
  if (!changelog.value) {
    return undefined;
  }
  const { instance, database } = extractDatabaseResourceName(
    changelog.value.name
  );
  return {
    project: project.value.name,
    instance,
    database,
    changelogId: extractChangelogUID(changelog.value.name),
  };
});

const show = computed(() => {
  return changelog.value !== undefined;
});
</script>
