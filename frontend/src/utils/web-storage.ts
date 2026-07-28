export class WebStorageHelper {
  storage: Storage;
  keyPrefix: string;

  constructor(keyPrefix: string, storage = localStorage) {
    this.keyPrefix = keyPrefix;
    this.storage = storage;
  }

  save<T>(key: string, value: T) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      const json = JSON.stringify(value);
      this.storage.setItem(fullKey, json);
    } catch {
      // nothing
    }
  }

  load<T>(key: string, fallbackValue: T) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      const json = this.storage.getItem(fullKey) || "";
      return JSON.parse(json) as T;
    } catch {
      return fallbackValue;
    }
  }

  remove(key: string) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      this.storage.removeItem(fullKey);
    } catch {
      // nothing
    }
  }

  keys(): string[] {
    const { length } = this.storage;
    const keys: string[] = [];
    for (let i = 0; i < length; i++) {
      const key = this.storage.key(i);
      if (key && key.startsWith(this.keyPrefix)) {
        keys.push(key);
      }
    }
    return keys;
  }

  clear() {
    const keys = this.keys();
    keys.forEach((key) => {
      this.storage.removeItem(key);
    });
  }
}
