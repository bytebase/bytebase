<template>
  <div class="mx-auto">
    <FeatureAttention
      v-if="!hasSchemaReviewPolicyFeature"
      custom-class="mt-5"
      feature="bb.feature.schema-review-policy"
      :description="
        $t('subscription.features.bb-feature-schema-review-policy.desc')
      "
    />
    <div v-if="store.reviewPolicyList.length > 0" class="space-y-6 my-5">
      <div class="flex items-center justify-end" v-if="hasPermission">
        <button
          type="button"
          class="btn-primary ml-3 inline-flex justify-center py-2 px-4"
          @click.prevent="goToCreationView"
        >
          {{ $t("schema-review-policy.create-policy") }}
        </button>
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
  featureToRef,
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

const hasSchemaReviewPolicyFeature = featureToRef(
  "bb.feature.schema-review-policy"
);

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
