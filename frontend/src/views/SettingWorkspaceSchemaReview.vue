<template>
  <div class="mx-auto">
    <div v-if="store.reviewPolicyList.length > 0" class="space-y-6 my-5">
      <div class="flex justify-start mt-4" v-if="hasPermission">
        <div class="flex flex-col items-center w-28">
          <button
            class="btn-icon-primary p-3"
            @click.prevent="goToCreationView"
          >
            <heroicons-outline:plus-sm class="w-6 h-6" />
          </button>
          <h3 class="mt-1 text-base font-normal text-main tracking-tight">
            {{ $t("schema-review-policy.add-review") }}
          </h3>
        </div>
      </div>
      <template v-for="review in store.reviewPolicyList" :key="review.id">
        <SchemaReviewCard :review-policy="review" @click="onClick" />
      </template>
    </div>
    <template v-else>
      <SchemaReviewEmptyView @create="goToCreationView" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, watchEffect } from "vue";
import { useRouter } from "vue-router";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useSchemaSystemStore,
  useCurrentUser,
} from "@/store";
import { DatabaseSchemaReviewPolicy } from "@/types/schemaSystem";
import { schemaReviewPolicySlug, isDBAOrOwner } from "@/utils";

const router = useRouter();
const store = useSchemaSystemStore();
const ROUTE_NAME = "setting.workspace.schema-review-policy";
const currentUser = useCurrentUser();
const { t } = useI18n();

watchEffect(() => {
  store.fetchReviewPolicyList();
});

const hasPermission = computed(() => {
  return isDBAOrOwner(currentUser.value.role);
});

const goToCreationView = () => {
  if (hasPermission.value) {
    router.push({
      name: `${ROUTE_NAME}.create`,
    });
  } else {
    pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-review-policy.no-permission"),
    });
  }
};

const onClick = (review: DatabaseSchemaReviewPolicy) => {
  router.push({
    name: `${ROUTE_NAME}.detail`,
    params: {
      schemaReviewPolicySlug: schemaReviewPolicySlug(review),
    },
  });
};
</script>
