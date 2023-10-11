<template>
  <Drawer :show="show" :close-on-esc="true" @close="$emit('close')">
    <DrawerContent
      :title="title"
      style="width: 75vw; max-width: calc(100vw - 8rem)"
    >
      <div class="h-full flex flex-col gap-y-2">
        <div class="w-full flex flex-row justify-between items-center gap-2">
          <BranchBaseline v-if="branch" :branch="branch" />
        </div>
        <SchemaDesignEditor
          v-if="editorBindings"
          class="flex-1"
          :readonly="true"
          :project="editorBindings.project"
          :branch="editorBindings.branch"
        />
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import SchemaDesignEditor from "@/components/Branch/SchemaDesignEditor.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { useSchemaDesignStore } from "@/store/modules/schemaDesign";
import BranchBaseline from "./BranchBaseline.vue";

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
    branch: branch.value,
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
