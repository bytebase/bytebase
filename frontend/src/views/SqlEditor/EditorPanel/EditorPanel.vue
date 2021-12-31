<template>
  <div class="editor-pane h-full">
    <EditorActions />

    <QueryEditor />
  </div>
</template>

<script lang="ts" setup>
import EditorActions from "./EditorAction.vue";
import QueryEditor from "./QueryEditor.vue";
</script>
