import { create } from "@bufbuild/protobuf";
import { computed, type Ref, ref, watch } from "vue";
import { releaseServiceClientConnect } from "@/connect";
import { ListReleaseCategoriesRequestSchema } from "@/types/proto-es/v1/release_service_pb";

export const useReleaseCategories = (projectName: Ref<string>) => {
  const categories = ref<string[]>([]);
  const loading = ref(false);

  const fetchCategories = async () => {
    if (!projectName.value) {
      categories.value = [];
      return;
    }

    loading.value = true;
    try {
      const request = create(ListReleaseCategoriesRequestSchema, {
        parent: projectName.value,
      });
      const response =
        await releaseServiceClientConnect.listReleaseCategories(request);
      categories.value = response.categories;
    } catch (error) {
      console.error("Failed to fetch release categories:", error);
      categories.value = [];
    } finally {
      loading.value = false;
    }
  };

  watch(
    () => projectName.value,
    () => {
      fetchCategories();
    },
    { immediate: true }
  );

  return {
    categories: computed(() => categories.value),
    loading: computed(() => loading.value),
    refresh: fetchCategories,
  };
};
