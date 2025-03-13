<template>
  <component
    :is="link ? 'router-link' : tag"
    v-bind="bindings"
    :class="link && !plain && 'normal-link'"
  >
    {{ email }}
  </component>
</template>

<script setup lang="ts">
import { computedAsync } from "@vueuse/core";
import { computed } from "vue";
import { useUserStore } from "@/store";

const props = withDefaults(
  defineProps<{
    email: string;
    tag?: string;
    link?: boolean;
    plain?: boolean;
  }>(),
  {
    tag: "span",
    link: true,
    plain: false,
  }
);

const store = useUserStore();

const user = computedAsync(() => {
  return store.getOrFetchUserByIdentifier(props.email);
});

const bindings = computed(() => {
  if (props.link) {
    const to = user.value ? `/users/${user.value.email}` : "/404";
    return {
      to,
      activeClass: "",
      exactActiveClass: "",
      onClick: (e: MouseEvent) => {
        e.stopPropagation();
      },
    };
  }
  return {};
});
</script>
