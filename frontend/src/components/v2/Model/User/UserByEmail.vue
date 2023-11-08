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
import { computed } from "vue";
import { useUserStore } from "@/store";
import { getUserEmailFromIdentifier } from "@/store/modules/v1/common";

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

const user = computed(() => {
  return store.getUserByEmail(getUserEmailFromIdentifier(props.email));
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
