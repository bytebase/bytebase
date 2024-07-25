<template>
  <div class="space-y-4 divide-y divide-block-border">
    <BBAttention type="info">
      <div class="textinfolabel">
        {{ $t("gitops.setting.add-git-provider.description") }}
        <LearnMoreLink
          url="https://www.bytebase.com/docs/vcs-integration/add-git-provider?source=console"
          class="text-sm"
        />
      </div>
    </BBAttention>
    <div class="pt-4">
      <VCSSetupWizard />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { BBAttention } from "@/bbkit";
import LearnMoreLink from "@/components/LearnMoreLink.vue";
import VCSSetupWizard from "@/components/VCS/VCSSetupWizard.vue";
</script>
