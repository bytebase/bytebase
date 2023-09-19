<template>
  <CreateDatabaseGuide
    v-if="shouldShowCreateDatabaseGuide"
  ></CreateDatabaseGuide>
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import {
  databaseServiceClient,
  instanceServiceClient,
  projectServiceClient,
} from "@/grpcweb";
import { useCurrentUserV1, useOnboardingGuideStore } from "@/store";
import { UNKNOWN_USER_NAME } from "@/types";
import { isDev, isOwner } from "@/utils";
import CreateDatabaseGuide from "./OnboardingGuides/CreateDatabaseGuide.vue";

const currentUserV1 = useCurrentUserV1();
const guideStore = useOnboardingGuideStore();

const shouldShowCreateDatabaseGuide = ref(false);

const route = useRoute();
const router = useRouter();

onMounted(async () => {
  await router.isReady();
  if (currentUserV1.value.name !== UNKNOWN_USER_NAME) {
    shouldShowCreateDatabaseGuide.value =
      await checkShouldShowCreateDatabaseGuide();
    if (shouldShowCreateDatabaseGuide.value) {
      guideStore.setGuideName("create-database");
    }
  }
});

const checkShouldShowCreateDatabaseGuide = async () => {
  // Do not show guide in dev mode and `noguide` flag in query.
  if (isDev() && route.query.noguide) {
    return false;
  }

  // Show create database guide when user is owner and no data at all.
  if (isOwner(currentUserV1.value.userRole)) {
    // Fetch data directly instead of useStore to prevent data from being cached in store.
    const { instances } = await instanceServiceClient.listInstances({});
    const { projects } = await projectServiceClient.searchProjects({});
    const { databases } = await databaseServiceClient.searchDatabases({
      parent: "instances/-",
    });
    if (
      instances.length === 0 &&
      // We have a default project so the length should be 1 not 0.
      projects.length === 1 &&
      databases.length === 0
    ) {
      return true;
    }
  }

  return false;
};

watch(
  () => currentUserV1.value.name,
  async (name) => {
    try {
      // Check should show guide only when user is logged in.
      if (name !== UNKNOWN_USER_NAME) {
        shouldShowCreateDatabaseGuide.value =
          await checkShouldShowCreateDatabaseGuide();
        if (shouldShowCreateDatabaseGuide.value) {
          guideStore.setGuideName("create-database");
        }
      }
    } catch (error) {
      // When the data requests failed in onboarding guide checking,
      // we just need to log it instead of notify.
      console.error(error);
    }
  }
);

watch(
  guideStore,
  async () => {
    if (!guideStore.guideName) {
      shouldShowCreateDatabaseGuide.value = false;
    }
  },
  {
    immediate: true,
  }
);
</script>
