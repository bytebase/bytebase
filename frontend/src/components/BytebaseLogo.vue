<template>
  <div
    class="flex-shrink-0 max-w-44 flex items-center overflow-hidden"
    @click="recordRedirect"
  >
    <component
      :is="component"
      :to="{
        name: redirect,
      }"
      class="h-full w-full select-none flex flex-row justify-center items-center"
    >
      <img
        v-if="customBrandingLogo"
        :src="customBrandingLogo"
        alt="branding logo"
        class="h-full object-contain"
      />

      <img
        v-else
        class="h-8 md:h-10 w-auto object-contain"
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
