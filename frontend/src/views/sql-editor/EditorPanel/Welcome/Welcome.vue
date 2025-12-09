<template>
  <div
    class="w-full flex-1 flex flex-col items-center justify-center gap-y-4"
  >
    <BytebaseLogo />

    <div
      class="flex items-center flex-wrap gap-4"
    >
      <Button
        v-if="showCreateInstanceButton"
        type="default"
        class="px-4!"
        @click="gotoInstanceCreatePage"
      >
        <template #icon>
          <LayersIcon :stroke-width="1.5" class="w-8 h-8" />
        </template>
        {{ $t("sql-editor.add-a-new-instance") }}
      </Button>
      <Button
        v-if="showConnectButton"
        secondary
        type="primary"
        class="px-4!"
        @click="changeConnection"
      >
        <template #icon>
          <LinkIcon :stroke-width="1.5" class="w-8 h-8" />
        </template>
        {{ $t("sql-editor.connect-to-a-database") }}
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { LayersIcon, LinkIcon } from "lucide-vue-next";
import { storeToRefs } from "pinia";
import { computed } from "vue";
import { useRouter } from "vue-router";
import BytebaseLogo from "@/components/BytebaseLogo.vue";
import { INSTANCE_ROUTE_DASHBOARD } from "@/router/dashboard/workspaceRoutes";
import { useProjectV1Store, useSQLEditorStore } from "@/store";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";
import { useSQLEditorContext } from "../../context";
import Button from "./Button.vue";

const { showConnectionPanel, asidePanelTab } = useSQLEditorContext();
const router = useRouter();
const projectStore = useProjectV1Store();
const { project: projectName } = storeToRefs(useSQLEditorStore());

const project = computed(() =>
  projectStore.getProjectByName(projectName.value)
);

const showCreateInstanceButton = computed(() => {
  return hasWorkspacePermissionV2("bb.instances.create");
});

const showConnectButton = computed(() =>
  hasProjectPermissionV2(project.value, "bb.sql.select")
);

const changeConnection = () => {
  asidePanelTab.value = "SCHEMA";
  showConnectionPanel.value = true;
};

const gotoInstanceCreatePage = () => {
  router.push({
    name: INSTANCE_ROUTE_DASHBOARD,
    hash: `#add`,
  });
};
</script>
