<template>
  <CreateDatabaseGuide
    v-if="shouldShowCreateDatabaseGuide"
  ></CreateDatabaseGuide>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";
import {
  useCurrentUser,
  useDatabaseStore,
  useOnboardingGuideStore,
} from "@/store";
import { UNKNOWN_ID } from "@/types";
import { isOwner } from "@/utils";
import CreateDatabaseGuide from "./OnboardingGuides/CreateDatabaseGuide.vue";

const guideStore = useOnboardingGuideStore();
const currentUser = useCurrentUser();

const shouldShowCreateDatabaseGuide = ref(false);

const checkShouldShowCreateDatabaseGuide = async () => {
  // Show create database guide when user is owner and no database data at all.
  if (isOwner(currentUser.value.role)) {
    const databaseList = await useDatabaseStore().fetchDatabaseList();
    if (databaseList.length === 0) {
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
      guideStore.setGuideName("create-database-guide");
    }
  },
  {
    immediate: true,
  }
);
</script>
