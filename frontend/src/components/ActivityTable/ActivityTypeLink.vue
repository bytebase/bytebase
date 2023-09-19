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
import { head } from "lodash-es";
import { computed, PropType } from "vue";
import { LogEntity, LogEntity_Action } from "@/types/proto/v1/logging_service";
import {
  ActivityProjectRepositoryPushPayload,
  ActivityProjectDatabaseTransferPayload,
} from "../../types";
import { Link } from "./types";

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
      const commit =
        head(payload.pushEvent.commits) ?? payload.pushEvent.fileCommit;
      if (commit && commit.id && commit.url) {
        return {
          title: commit.id.substring(0, 7),
          path: commit.url,
          external: true,
        };
      }
      // Downgrade for legacy data.
      return undefined;
    }
    case LogEntity_Action.ACTION_PROJECT_DATABASE_TRANSFER: {
      const payload = JSON.parse(
        activity.payload
      ) as ActivityProjectDatabaseTransferPayload;
      return {
        title: payload.databaseName,
        path: `/db/${payload.databaseId}`,
        external: false,
      };
    }
  }
  return undefined;
});
</script>
