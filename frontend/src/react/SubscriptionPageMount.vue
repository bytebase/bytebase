<template>
  <ReactPageMount page="SubscriptionPage" v-bind="pageProps" />
  <InstanceAssignment
    :show="showInstanceAssignmentDrawer"
    @dismiss="showInstanceAssignmentDrawer = false"
  />
</template>

<script lang="ts" setup>
import { ref } from "vue";
import InstanceAssignment from "@/components/InstanceAssignment.vue";
import { ENTERPRISE_INQUIRE_LINK } from "@/types";
import ReactPageMount from "./ReactPageMount.vue";

const showInstanceAssignmentDrawer = ref(false);

const pageProps = {
  onRequireEnterprise: () => {
    window.open(ENTERPRISE_INQUIRE_LINK, "_blank");
  },
  onManageInstanceLicenses: () => {
    showInstanceAssignmentDrawer.value = true;
  },
};
</script>
