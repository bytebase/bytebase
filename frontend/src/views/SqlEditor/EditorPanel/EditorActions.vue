<template>
  <div class="sqleditor-editor-actions">
    <div class="actions-left">
        <n-button type="primary">
        <mdi:play class="h-5 w-5" /> Run (⌘+⏎)
      </n-button>
    </div>
    <div class="actions-right space-x-2">
      <n-button secondary strong type="primary">
        <carbon:save class="h-5 w-5" /> &nbsp; Save (⌘+S)
      </n-button>
      <n-button>
        <carbon:share class="h-5 w-5" /> &nbsp; Share (⌘+S)
      </n-button>
    </div>
  </div>
</template>

<style scoped>
.sqleditor-editor-actions {
  @apply w-full flex justify-between items-center p-2;
}
</style>
