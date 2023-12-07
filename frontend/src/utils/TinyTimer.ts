import { isRelease } from "./util";

type TimerEntry = {
  tag: string;
  time: number; // total consumed times, in milliseconds
  count: number; // measure how many times this tag is called
  loops: number; // measure how many loops in this tag
};

const now = () => {
  return performance.now();
};

const emptyTimerEntry = (tag: string): TimerEntry => {
  return {
    tag,
    time: 0,
    count: 0,
    loops: 0,
  };
};

export class TinyTimer<T extends string = string> {
  private entires: Record<T, TimerEntry> = {} as any;
  private begins: Record<T, number> = {} as any;
  constructor() {}
  begin(tag: T) {
    if (isRelease()) return;

    this.begins[tag] = now();
  }
  end(tag: T, loops = 1) {
    if (isRelease()) return;

    const begin = this.begins[tag];
    // begin timestamp not found, give up
    if (!begin) return;
    const end = now();
    const elapsed = end - begin;
    const entry =
      this.entires[tag] ?? (this.entires[tag] = emptyTimerEntry(tag));
    entry.count++;
    entry.loops += loops;
    entry.time += elapsed;
  }
  print(tag: T) {
    if (isRelease()) return;

    const entry = this.entires[tag];
    if (!entry) return;
    console.debug(JSON.stringify(entry, null, "  "));
  }
  printAll() {
    if (isRelease()) return;

    const tags = Object.keys(this.entires);
    tags.forEach((tag) => this.print(tag as T));
  }
}
