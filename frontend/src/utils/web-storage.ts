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
      localStorage.setItem(fullKey, json);
    } catch {
      // nothing
    }
  }

  load<T>(key: string, fallbackValue: T) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      const json = localStorage.getItem(fullKey) || "";
      return JSON.parse(json) as T;
    } catch {
      return fallbackValue;
    }
  }

  remove(key: string) {
    const fullKey = `${this.keyPrefix}.${key}`;
    try {
      localStorage.removeItem(fullKey);
    } catch {
      // nothing
    }
  }

  keys(): string[] {
    const { length } = localStorage;
    const keys: string[] = [];
    for (let i = 0; i < length; i++) {
      const key = localStorage.key(i);
      if (key && key.startsWith(this.keyPrefix)) {
        keys.push(key);
      }
    }
    return keys;
  }

  clear() {
    const keys = this.keys();
    keys.forEach((key) => {
      localStorage.removeItem(key);
    });
  }
}
