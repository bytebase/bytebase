<template>
  <div class="w-full">
    <BBAttention
      :style="'WARN'"
      :title="$t('two-factor.messages.2fa-required')"
      side="CENTER"
    />
    <div class="w-full p-2 sm:p-8 sm:px-16">
      <TwoFactorSetup />
    </div>
  </div>
</template>

<script lang="ts" setup>
import TwoFactorSetup from "./TwoFactorSetup.vue";
</script>
