<template>
  <div class="flex flex-col gap-y-4 flex-1 overflow-hidden">
    <RawSQLEditor
      v-if="sheet"
      v-model:statement="statement"
      :readonly="false"
      class="flex-1 overflow-hidden relative"
      @upload="handleUploadEvent"
    />
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useLocalSheetStore } from "@/store";
import { getSheetStatement, readFileAsync, setSheetStatement } from "@/utils";
import RawSQLEditor from "../../RawSQLEditor";
import { useAddChangeContext } from "../context";

const { changeFromRawSQL: change } = useAddChangeContext();

const sheet = computed(() => {
  return useLocalSheetStore().getOrCreateSheetByName(change.value.sheet);
});

const statement = computed({
  get() {
    return getSheetStatement(sheet.value);
  },
  set(statement) {
    setSheetStatement(sheet.value, statement);
  },
});

const handleUploadEvent = async (e: Event) => {
  try {
    const { filename, content } = await readFileAsync(e, 100);
    statement.value = content;
    sheet.value.title = filename;
  } finally {
    // Nothing
  }
};
</script>
