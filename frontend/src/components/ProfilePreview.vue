<template>
  <component
    :is="link ? 'router-link' : 'div'"
    class="px-4 py-3 menu-item w-48"
    role="menuitem"
    v-bind="bindings"
    @click="$emit('click')"
  >
    <p class="text-sm flex justify-between gap-x-2">
      <span class="text-main font-medium truncate text-ellipsis">
        {{ currentUserV1.title }}
      </span>
    </p>
    <p class="text-sm text-control truncate text-ellipsis">
      {{ currentUserV1.email }}
    </p>
  </component>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useCurrentUserV1 } from "@/store";
import { autoProfileLink } from "@/utils";

const props = withDefaults(
  defineProps<{
    link?: boolean;
  }>(),
  {
    link: true,
  }
);

defineEmits<{
  (event: "click"): void;
}>();

const router = useRouter();
const currentUserV1 = useCurrentUserV1();

const bindings = computed(() => {
  if (!props.link) return {};
  return {
    to: autoProfileLink(router),
  };
});
</script>
