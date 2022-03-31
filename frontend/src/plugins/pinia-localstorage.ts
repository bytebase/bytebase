/* eslint-disable no-console */
import { watch } from "vue";
import { PiniaPluginContext } from "pinia";
import { useLocalStorage } from "@vueuse/core";

export interface PersistOptions {
  enabled: boolean;
  strategy?: string;
}

type Store = PiniaPluginContext["store"];
type PartialState = Partial<Store["$state"]>;

declare module "pinia" {
  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  export interface DefineStoreOptionsBase<S extends StateTree, Store> {
    persist?: PersistOptions;
  }
}

const updateStorage = (store: Store) => {
  const storage = localStorage;
  const storeKey = store.$id;

  storage.setItem(storeKey, JSON.stringify(store.$state));
};

export default ({ options, store }: PiniaPluginContext): void => {
  if (
    options.persist?.enabled &&
    options.persist?.strategy === "localStorage"
  ) {
    const storage = localStorage;
    const storeKey = store.$id;
    const storageResult = storage.getItem(storeKey);

    if (storageResult) {
      const parsedResult = JSON.parse(storageResult) as PartialState;
      const reactiveStore = useLocalStorage(storeKey, parsedResult, {
        listenToStorageChanges: true,
      });
      store.$patch(parsedResult);
      updateStorage(store);

      // added sync back to store when localStorage value change
      watch(
        () => reactiveStore.value,
        (newValue) => {
          store.$patch(newValue);
        },
        { deep: true }
      );
    }

    watch(
      () => store.$state,
      () => {
        updateStorage(store);
      },
      { deep: true }
    );
  }
};
