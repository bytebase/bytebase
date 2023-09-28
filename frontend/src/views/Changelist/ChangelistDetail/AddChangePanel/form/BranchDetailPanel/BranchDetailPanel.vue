<template>
  <Drawer :show="show" :close-on-esc="false" @close="$emit('close')">
    <DrawerContent
      :title="title"
      style="width: 60vw; max-width: calc(100vw - 8rem)"
    >
      <SchemaEditorV1
        v-if="editorBindings"
        :readonly="true"
        :project="editorBindings.project"
        :resource-type="'branch'"
        :databases="editorBindings.databases"
        :branches="editorBindings.branches"
      />
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import SchemaEditorV1 from "@/components/SchemaEditorV1/index.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";

const props = defineProps<{
  branchName?: string;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const title = ref(t("common.branch"));

const branch = computed(() => {
  const { branchName } = props;
  if (!branchName) {
    return undefined;
  }
  return useSchemaDesignStore().getSchemaDesignByName(branchName);
});

const editorBindings = computed(() => {
  if (!branch.value) {
    return undefined;
  }

  const db = useDatabaseV1Store().getDatabaseByName(
    branch.value.baselineDatabase
  );
  const project = db.projectEntity;

  return {
    readonly: true,
    project,
    resourceType: "branch",
    databases: [db],
    branches: [branch.value],
  };
});

const show = computed(() => {
  return branch.value !== undefined;
});

watch(branch, (branch) => {
  if (branch) {
    title.value = branch.title;
  }
});
</script>
