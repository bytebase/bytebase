<template>
  <div class="flex flex-row items-center flex-wrap gap-2">
    <NTag v-for="role in sortRoles(uniq(roles))" :key="role">
      {{ displayRoleTitle(role) }}
    </NTag>
  </div>
</template>

<script lang="ts" setup>
import { uniq } from "lodash-es";
import { NTag } from "naive-ui";
import { displayRoleTitle, sortRoles } from "@/utils";

defineProps<{
  roles: string[];
}>();
</script>
