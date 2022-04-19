<template>
  <div class="bg-white rounded-lg px-5 py-6 sm:px-6">
    <div
      class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex justify-center items-center"
    >
      <div class="text-center flex flex-col justify-center items-center">
        <img src="../../assets/illustration/no-data.webp" class="w-52" />
        <h3 class="mt-2 font-medium text-gray-900">
          {{ $t("schema-review-policy.no-policy-set") }}
        </h3>
        <div class="mt-6">
          <button
            type="button"
            class="inline-flex items-center px-4 py-2 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            @click="$emit('create')"
          >
            {{ $t("schema-review-policy.add-review") }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
defineEmits(["create"]);
</script>
