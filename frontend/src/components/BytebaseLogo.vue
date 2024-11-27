<template>
  <div
    class="flex-shrink-0 w-44 min-h-[4rem] flex items-center overflow-y-hidden"
    @click="recordRedirect"
  >
    <component
      :is="component"
      :to="{
        name: redirect,
      }"
      class="w-full select-none flex flex-row justify-center items-center"
      active-class=""
      exact-active-class=""
    >
      <img
        v-if="customBrandingLogo"
        :src="customBrandingLogo"
        class="w-auto max-w-full my-3"
      />

      <img
        v-else
        class="h-8 md:h-10 w-auto"
        src="@/assets/logo-full.svg"
        alt="Bytebase"
      />
    </component>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useRecentVisit } from "@/router/useRecentVisit";
import { useActuatorV1Store } from "@/store/modules/v1/actuator";

const props = withDefaults(
  defineProps<{
    redirect?: string;
  }>(),
  {
    redirect: "",
  }
);

const component = computed(() => (props.redirect ? "router-link" : "span"));
const { record } = useRecentVisit();
const router = useRouter();

const customBrandingLogo = computed((): string => {
  return useActuatorV1Store().brandingLogo;
});

const recordRedirect = () => {
  if (!props.redirect) {
    return;
  }
  const route = router.resolve({
    name: props.redirect,
  });
  record(route.fullPath);
};
</script>
