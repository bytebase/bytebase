<template>
  <ChatPanel v-if="openAIKey" />
</template>

<script lang="ts" setup>
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import type { SQLEditorConnection } from "@/types";
import { useAIContext } from "../logic";

const [{ default: ChatPanel }] = await Promise.all([import("./ChatPanel.vue")]);
const { events, openAIKey } = useAIContext();

const emit = defineEmits<{
  (
    event: "apply-statement",
    statement: string,
    conn: SQLEditorConnection,
    run: boolean
  ): void;
}>();

const { connection } = useConnectionOfCurrentSQLEditorTab();

events.on("apply-statement", ({ statement, run }) => {
  emit("apply-statement", statement, connection.value, run);
});
</script>
