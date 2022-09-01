<template>
  <div :class="`tag ${method}`">{{ method }}</div>
</template>

<script lang="ts" setup>
defineProps({
  method: {
    required: true,
    type: String,
  },
});
</script>

<style scoped>
.tag {
  @apply h-6 text-white text-xs px-2 rounded whitespace-nowrap inline-flex items-center;
}

.GET {
  border: 1px solid #0f6ab4;
  color: #0f6ab4;
}
.POST {
  border: 1px solid #10a54a;
  color: #10a54a;
}
.PUT {
  border: 1px solid #c5862b;
  color: #c5862b;
}
.DELETE {
  border: 1px solid #a41e22;
  color: #a41e22;
}
.PATCH {
  border: 1px solid #d38042;
  color: #d38042;
}
.HEAD {
  border: 1px solid #ffd20f;
  color: #ffd20f;
}
</style>
