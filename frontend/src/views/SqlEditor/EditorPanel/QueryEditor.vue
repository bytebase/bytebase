<template>
  <MonacoEditor v-model="sqlCode" @change="handleChange" />
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { debounce } from "lodash-es";
import { useStore } from "vuex";
import { useVuex } from "@vueblocks/vue-use-vuex";

const sqlCode = ref("");
const store = useStore();
const { useActions } = useVuex("sqlEditor", store);

const { setSqlEditorState } = useActions(["setSqlEditorState"]) as any;

const handleChange = debounce((value: string) => {
  console.log("handleChange", value);
  setSqlEditorState({
    queryStatement: value,
  });
}, 300);
</script>
