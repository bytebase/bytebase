<template>
  <Drawer :show="show" @close="$emit('close')">
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
          :hide-s-q-l-check-button="true"
        />
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script setup lang="ts">
import { asyncComputed } from "@vueuse/core";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import SchemaDesignEditor from "@/components/Branch/SchemaDesignEditor.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { useDatabaseV1Store } from "@/store";
import { useBranchStore } from "@/store/modules/branch";
import BranchBaseline from "./BranchBaseline.vue";

const props = defineProps<{
  branchName?: string;
}>();

defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const title = ref(t("common.branch"));

const branch = asyncComputed(async () => {
  const { branchName } = props;
  if (!branchName) {
    return undefined;
  }
  return await useBranchStore().fetchBranchByName(
    branchName,
    true /* useCache */
  );
}, undefined);

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
