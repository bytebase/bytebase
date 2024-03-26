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
import type { PropType } from "vue";
import { computed } from "vue";
import type { LogEntity } from "@/types/proto/v1/logging_service";
import { LogEntity_Action } from "@/types/proto/v1/logging_service";
import type { ActivityProjectRepositoryPushPayload } from "../../types";
import { issueSlug } from "../../utils/slug";
import type { Link } from "./types";

const props = defineProps({
  activity: {
    type: Object as PropType<LogEntity>,
    required: true,
  },
});

const link = computed((): Link | undefined => {
  const { activity } = props;
  switch (activity.action) {
    case LogEntity_Action.ACTION_PROJECT_REPOSITORY_PUSH: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityProjectRepositoryPushPayload;
      if (payload.issueId && payload.issueName) {
        return {
          title: `issue/${payload.issueId}`,
          path: `/issue/${issueSlug(payload.issueName!, payload.issueId!)}`,
          external: false,
        };
      }
      break;
    }
  }
  return undefined;
});
</script>
