<template>
  <div class="space-y-4 h-full flex flex-col">
    <BBAttention type="info">
      <div class="textinfolabel">
        {{ $t("gitops.setting.add-git-provider.description") }}
        <LearnMoreLink
          url="https://www.bytebase.com/docs/vcs-integration/add-git-provider?source=console"
          class="text-sm"
        />
      </div>
    </BBAttention>
    <VCSSetupWizard class="flex-1" :show-cancel="true" />
  </div>
</template>

<script lang="ts" setup>
import { BBAttention } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import VCSSetupWizard from "@/components/VCS/VCSSetupWizard.vue";
</script>
