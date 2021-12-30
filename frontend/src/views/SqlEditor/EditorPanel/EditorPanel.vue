<template>
  <div class="editor-pane h-full">
    <EditorActions />

    <QueryEditor />
  </div>
</template>

<script lang="ts" setup>
import EditorActions from "./EditorActions.vue";
import QueryEditor from "./QueryEditor.vue";
</script>
