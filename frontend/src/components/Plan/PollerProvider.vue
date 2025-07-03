<template>
  <slot />
</template>

<script lang="ts" setup>
import { provideResourcePoller } from "./logic/poller";

provideResourcePoller();
</script>
