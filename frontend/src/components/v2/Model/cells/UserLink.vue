<template>
  <component
    :is="isLink ? 'router-link' : 'div'"
    class="truncate max-w-[10em]"
    :class="isLink && 'normal-link'"
    :to="`/users/${email}`"
  >
    <HighlightLabelText
      :keyword="keyword ?? ''"
      :text="title"
    />
  </component>
</template>

<script lang="tsx" setup>
import { computed } from "vue";
import { HighlightLabelText } from "@/components/v2";
import { hasWorkspacePermissionV2 } from "@/utils";

const props = withDefaults(
  defineProps<{
    email: string;
    title: string;
    keyword?: string;
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

const isLink = computed(
  () => props.link && hasWorkspacePermissionV2("bb.users.get")
);
</script>