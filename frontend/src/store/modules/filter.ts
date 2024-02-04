import { defineStore } from "pinia";
import { ref } from "vue";
import { useRouter } from "vue-router";

interface Filter {
  // project mainly using to filter databases in SQL Editor.
  // If specified, only databases in the project will be shown.
  // Format: "projects/{project}"
  project?: string;
  // database mainly using to specify the selected database.
  // Using in SQL Editor and other database related pages.
  // Format: "instances/{instance}/databases/{database}"
  database?: string;
}

export const useFilterStore = defineStore("filter", () => {
  const router = useRouter();
  const filter = ref<Filter>({});

  // Initial filter with route query immediately.
  // And it should not be updated when route changed later except the page is reloaded.
  const route = router.currentRoute.value;
  if (route.query.filter && typeof route.query.filter === "string") {
    try {
      filter.value = JSON.parse(route.query.filter);
    } catch (error) {
      console.error("Failed to parse filter", route.query.filter);
    }
  }

  return {
    filter,
  };
});
