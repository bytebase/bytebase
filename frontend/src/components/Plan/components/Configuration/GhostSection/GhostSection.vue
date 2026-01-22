<template>
  <div class="flex flex-col gap-1">
    <OptionRow>
      <template #label>
        {{ $t("task.online-migration.self") }}
      </template>
      <template #tooltip>
        <i18n-t
          tag="p"
          keypath="issue.migration-mode.online.description"
          class="whitespace-pre-line"
        >
          <template #link>
            <LearnMoreLink
              url="https://docs.bytebase.com/change-database/online-schema-migration-for-mysql?source=console"
              color="light"
            />
          </template>
        </i18n-t>
      </template>
      <template #suffix>
        <NButton
          v-if="enabled && allowChange"
          tag="div"
          size="tiny"
          style="--n-padding: 0 5px"
          @click="showFlagsPanel = true"
        >
          <template #icon>
            <WrenchIcon class="w-4 h-4" />
          </template>
          {{ $t("task.online-migration.configure") }}
        </NButton>
      </template>
      <GhostSwitch />
    </OptionRow>

    <GhostFlagsPanel
      :show="showFlagsPanel"
      @update:show="(show) => (showFlagsPanel = show)"
    />
  </div>
</template>

<script lang="ts" setup>
import { WrenchIcon } from "lucide-vue-next";
import { NButton } from "naive-ui";
import { computed, ref } from "vue";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import { useSelectedSpec } from "../../SpecDetailView/context";
import { isGhostEnabled } from "../../StatementSection/directiveUtils";
import { useSpecSheet } from "../../StatementSection/useSpecSheet";
import OptionRow from "../OptionRow.vue";
import { useGhostSettingContext } from "./context";
import GhostFlagsPanel from "./GhostFlagsPanel.vue";
import GhostSwitch from "./GhostSwitch.vue";

const showFlagsPanel = ref(false);

const { allowChange } = useGhostSettingContext();
const { selectedSpec } = useSelectedSpec();
const { sheetStatement, sheetReady } = useSpecSheet(selectedSpec);

const enabled = computed(() => {
  if (!sheetReady.value) return false;
  return isGhostEnabled(sheetStatement.value);
});
</script>
