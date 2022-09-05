<template>
  <CreateDatabaseGuide
    v-if="shouldShowCreateDatabaseGuide"
  ></CreateDatabaseGuide>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";
import {
  useCurrentUser,
  useProjectStore,
  useInstanceStore,
  useDatabaseStore,
  useOnboardingGuideStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { isOwner } from "@/utils";
import CreateDatabaseGuide from "./OnboardingGuides/CreateDatabaseGuide.vue";

const currentUser = useCurrentUser();
const guideStore = useOnboardingGuideStore();

const shouldShowCreateDatabaseGuide = ref(false);

const checkShouldShowCreateDatabaseGuide = async () => {
  // Show create database guide when user is owner and no data at all.
  if (isOwner(currentUser.value.role)) {
    const instanceList = await useInstanceStore().fetchInstanceList();
    const projectList = await useProjectStore().fetchAllProjectList();
    const databaseList = await useDatabaseStore().fetchDatabaseList();

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

watch(
  currentUser,
  async () => {
    // Check should show guide only when user is logged in.
    if (currentUser.value.id !== UNKNOWN_ID) {
      shouldShowCreateDatabaseGuide.value =
        await checkShouldShowCreateDatabaseGuide();
      if (shouldShowCreateDatabaseGuide.value) {
        guideStore.setGuideName("create-database");
      }
    }
  },
  {
    immediate: true,
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
