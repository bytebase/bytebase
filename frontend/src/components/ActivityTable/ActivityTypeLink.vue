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

<script lang="ts">
import { computed, defineComponent, PropType } from "vue";
import { head } from "lodash-es";

import {
  Activity,
  ActivityProjectRepositoryPushPayload,
  ActivityProjectDatabaseTransferPayload,
} from "../../types";
import { Link } from "./types";

export default defineComponent({
  name: "ActivityTypeLink",
  props: {
    activity: {
      type: Object as PropType<Activity>,
      required: true,
    },
  },
  setup(props) {
    const link = computed((): Link | undefined => {
      const { activity } = props;
      switch (activity.type) {
        case "bb.project.repository.push": {
          const payload =
            activity.payload as ActivityProjectRepositoryPushPayload;
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
        case "bb.project.database.transfer": {
          const payload =
            activity.payload as ActivityProjectDatabaseTransferPayload;
          return {
            title: payload.databaseName,
            path: `/db/${payload.databaseId}`,
            external: false,
          };
        }
      }
      return undefined;
    });
    return { link };
  },
});
</script>
