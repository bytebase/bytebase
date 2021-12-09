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
import { Activity, ActivityProjectRepositoryPushPayload } from "../../types";
import { issueSlug } from "../../utils/slug";
import { Link } from "./types";

export default defineComponent({
  name: "ActivityCommentLink",
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
    return { link };
  },
});
</script>
