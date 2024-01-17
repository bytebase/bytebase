<template>
  <div class="space-y-4 divide-y divide-block-border">
    <div class="textinfolabel">
      <i18n-t keypath="gitops.setting.add-git-provider.description">
        <template #guide>
          <a
            href="https://bytebase.com/docs/vcs-integration/overview?source=console"
            target="_blank"
            class="normal-link"
          >
            {{ $t("common.detailed-guide") }}</a
          >
        </template>
      </i18n-t>
    </div>
    <div class="pt-4">
      <VCSSetupWizard />
    </div>
  </div>
</template>

<script lang="ts" setup>
import VCSSetupWizard from "@/components/VCS/VCSSetupWizard.vue";
</script>
