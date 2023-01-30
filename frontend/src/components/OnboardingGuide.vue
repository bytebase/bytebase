<template>
  <CreateDatabaseGuide
    v-if="shouldShowCreateDatabaseGuide"
  ></CreateDatabaseGuide>
</template>

<script lang="ts" setup>
import axios from "axios";
import { onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { useCurrentUser, useOnboardingGuideStore } from "@/store";
import { UNKNOWN_ID } from "@/types";
import { isDev, isOwner } from "@/utils";
import CreateDatabaseGuide from "./OnboardingGuides/CreateDatabaseGuide.vue";

const currentUser = useCurrentUser();
const guideStore = useOnboardingGuideStore();

const shouldShowCreateDatabaseGuide = ref(false);

const route = useRoute();
const router = useRouter();

onMounted(async () => {
  await router.isReady();
  if (currentUser.value.id !== UNKNOWN_ID) {
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
  if (isOwner(currentUser.value.role)) {
    // Fetch data directly instead of useStore to prevent data from being cached in store.
    const { data: instanceList } = (await axios.get(`/api/instance`)).data;
    const { data: projectList } = (
      await axios.get(`/api/project?user=${currentUser.value.id}`)
    ).data;
    const { data: databaseList } = (await axios.get(`/api/database`)).data;
    if (
      instanceList.length === 0 &&
      // We have a default project so the length should be 1 not 0.
      projectList.length === 1 &&
      databaseList.length === 0
    ) {
      return true;
    }
  }

  return false;
};

watch(currentUser, async () => {
  try {
    // Check should show guide only when user is logged in.
    if (currentUser.value.id !== UNKNOWN_ID) {
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
});

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
