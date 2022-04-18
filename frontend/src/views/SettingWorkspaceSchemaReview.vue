<template>
  <div class="mx-auto">
    <div v-if="store.reviewList.length > 0" class="space-y-6 my-5">
      <div class="flex justify-start mt-4">
        <div class="flex flex-col items-center w-28">
          <button
            class="btn-icon-primary p-3"
            @click.prevent="goToCreationView"
          >
            <heroicons-outline:plus-sm class="w-6 h-6" />
          </button>
          <h3 class="mt-1 text-base font-normal text-main tracking-tight">
            {{ $t("schema-review.add-review") }}
          </h3>
        </div>
      </div>
      <template v-for="review in store.reviewList" :key="review.id">
        <SchemaReviewCard :review="review" @click="onClick" />
      </template>
    </div>
    <template v-else>
      <SchemaReviewEmptyView @click="goToCreationView" />
    </template>
  </div>
</template>

<script lang="ts" setup>
import { useRouter } from "vue-router";
import { useSchemaSystemStore } from "@/store";
import { DatabaseSchemaReview } from "../types/schemaSystem";
import { schemaReviewSlug } from "../utils";

const router = useRouter();
const store = useSchemaSystemStore();
const ROUTE_NAME = "setting.workspace.schema-review";

const goToCreationView = () => {
  router.push({
    name: `${ROUTE_NAME}.create`,
  });
};

const onClick = (review: DatabaseSchemaReview) => {
  router.push({
    name: `${ROUTE_NAME}.detail`,
    params: {
      schemaReviewSlug: schemaReviewSlug(review),
    },
  });
};
</script>
