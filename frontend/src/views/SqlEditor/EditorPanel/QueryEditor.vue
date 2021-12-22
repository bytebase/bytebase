<template>
  <MonacoEditor v-model="sqlCode" />
</template>

<script lang="ts" setup>
import { ref } from 'vue'

const sqlCode = ref('')
</script>
