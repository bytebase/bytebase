<template>
  <div class="mx-auto w-full max-w-sm">
    <div>
      <img
        class="h-12 w-auto"
        src="../../assets/logo-full.svg"
        alt="Bytebase"
      />
      <h2 class="mt-6 text-3xl leading-9 font-extrabold text-main">
        {{ $t("auth.password-forget.title") }}
      </h2>
    </div>

    <div class="mt-8">
      <div class="mt-6">
        <BBAttention :style="`WARN`" :title="hint">
          <template v-if="actuatorStore.isSaaSMode && !hint" #title>
            <i18n-t
              tag="h3"
              class="text-sm font-medium text-yellow-800"
              keypath="auth.password-forget.cloud"
            >
              <template #link>
                <a
                  href="https://hub.bytebase.com/workspace"
                  target="_blank"
                  class="normal-link"
                >
                  Bytebase Hub
                </a>
              </template>
            </i18n-t>
          </template>
        </BBAttention>
      </div>
    </div>

    <div class="mt-6 relative">
      <div class="absolute inset-0 flex items-center" aria-hidden="true">
        <div class="w-full border-t border-control-border"></div>
      </div>
      <div class="relative flex justify-center text-sm">
        <router-link to="/auth" class="accent-link bg-white px-2">
          {{ $t("auth.password-forget.return-to-sign-in") }}
        </router-link>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import { useRoute } from "vue-router";
import { useActuatorV1Store } from "@/store";

const route = useRoute();
const { t } = useI18n();
const actuatorStore = useActuatorV1Store();

const hint = computed(() => {
  const hint = route.query.hint as string;
  if (hint) {
    return hint;
  }
  if (!actuatorStore.isSaaSMode) {
    return t("auth.password-forget.selfhost");
  }

  return "";
});
</script>
