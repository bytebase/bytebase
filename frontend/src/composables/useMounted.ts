import { onBeforeUnmount, onMounted, ref } from "vue";

export const useMounted = () => {
  const mounted = ref(false);
  onMounted(() => (mounted.value = true));
  onBeforeUnmount(() => (mounted.value = false));
  return mounted;
};
