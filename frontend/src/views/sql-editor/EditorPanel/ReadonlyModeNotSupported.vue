<template>
  <div class="w-full h-full flex flex-col items-center justify-center gap-y-2">
    <NoPermissionPlaceholder />
    <i18n-t
      class="textinfolabel flex items-center"
      keypath="sql-editor.allow-admin-mode-only"
      tag="div"
    >
      <template #instance>
        <InstanceV1Name :instance="instance" :link="false" />
      </template>
    </i18n-t>
    <AdminModeButton />
  </div>
</template>

<script lang="ts" setup>
import NoPermissionPlaceholder from "@/components/Permission/NoPermissionPlaceholder.vue";
import { InstanceV1Name } from "@/components/v2";
import { useConnectionOfCurrentSQLEditorTab } from "@/store";
import AdminModeButton from "../EditorCommon/AdminModeButton.vue";

const { instance } = useConnectionOfCurrentSQLEditorTab();
</script>
