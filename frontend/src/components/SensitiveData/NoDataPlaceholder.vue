<template>
  <div
    class="border-4 border-dashed border-gray-200 rounded-lg h-96 flex flex-col justify-center items-center"
  >
    <NoData
      :img-attrs="{
        class: '!w-auto max-h-[20vh]',
      }"
    />
    <slot />
  </div>
</template>

<script lang="ts" setup></script>
