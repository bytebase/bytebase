<template>
  <NCheckbox
    v-if="allowFormatOnSave"
    :checked="formatOnSave"
    size="small"
    @update:checked="updateFormatOnSave"
  >
    {{ $t("issue.format-on-save") }}
  </NCheckbox>
</template>

<script setup lang="ts">
import { NCheckbox } from "naive-ui";
import { computed } from "vue";
import { databaseForSpec } from "@/components/Plan/logic";
import { useUIStateStore } from "@/store";
import { useCurrentProjectV1 } from "@/store";
import { useInstanceV1EditorLanguage } from "@/utils";
import { useSelectedSpec } from "../../SpecDetailView/context";

const { project } = useCurrentProjectV1();
const selectedSpec = useSelectedSpec();
const uiStateStore = useUIStateStore();

const database = computed(() => {
  return databaseForSpec(project.value, selectedSpec.value);
});

const language = useInstanceV1EditorLanguage(
  computed(() => database.value.instanceResource)
);

const allowFormatOnSave = computed(() => language.value === "sql");

const formatOnSave = computed(() => uiStateStore.editorFormatStatementOnSave);

const updateFormatOnSave = (value: boolean) => {
  uiStateStore.setEditorFormatStatementOnSave(value);
};
</script>
