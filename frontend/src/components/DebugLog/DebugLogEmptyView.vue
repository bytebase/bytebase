<template>
  <div class="bg-white rounded-lg py-6">
    <div
      class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex justify-center items-center"
    >
      <div class="text-center flex flex-col justify-center items-center">
        <img src="../../assets/illustration/no-data.webp" class="w-52" />
        <h3 class="mt-2 font-medium text-gray-900">
          {{ $t("debug-log.no-logs") }}
        </h3>
      </div>
    </div>
  </div>
</template>
