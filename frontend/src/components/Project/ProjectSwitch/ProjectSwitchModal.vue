<template>
  <BBModal
    :close-on-esc="true"
    :mask-closable="true"
    :trap-focus="false"
    :show="show"
    :title="$t('project.select')"
    class="w-3xl max-w-full h-auto max-h-full"
    @close="$emit('dismiss')"
  >
    <ProjectSwitchContent
      @on-create="
        () => {
          showCreateDrawer = true;
        }
      "
    />
  </BBModal>

  <Drawer
    :auto-focus="true"
    :close-on-esc="true"
    :show="showCreateDrawer"
    @close="showCreateDrawer = false"
  >
    <ProjectCreatePanel @dismiss="showCreateDrawer = false" />
  </Drawer>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { BBModal } from "@/bbkit";
import ProjectCreatePanel from "@/components/Project/ProjectCreatePanel.vue";
import { Drawer } from "@/components/v2";
import ProjectSwitchContent from "./ProjectSwitchContent.vue";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  (event: "dismiss"): void;
}>();

const showCreateDrawer = ref<boolean>(false);
</script>
