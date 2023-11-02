<template>
  <template v-if="link">
    <a
      v-if="link.external"
      :href="link.path"
      target="_blank"
      class="normal-link flex flex-row items-center"
    >
      <span>{{ link.title }}</span>
      <heroicons-outline:external-link class="w-4 h-4" />
    </a>
    <router-link v-else class="normal-link" :to="link.path">
      {{ link.title }}
    </router-link>
  </template>
</template>

<script lang="ts" setup>
import { computed, PropType } from "vue";
import { LogEntity } from "@/types/proto/v1/logging_service";
import { getLinkFromActivity } from "./utils";

const props = defineProps({
  activity: {
    type: Object as PropType<LogEntity>,
    required: true,
  },
});

const link = computed(() => getLinkFromActivity(props.activity));
</script>
