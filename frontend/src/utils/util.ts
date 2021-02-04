import moment from "moment";

export function humanize(ts: number) {
  const time = moment.utc(ts);
  if (moment().year() == time.year()) {
    if (moment().dayOfYear() == time.dayOfYear()) {
      return time.format("HH:mm");
    }
    if (moment().diff(time, "days") < 3) {
      return time.format("MMM D HH:mm");
    }
    return time.format("MMM D");
  }
  return time.format("MMM D YYYY");
}

export function urlfy(str: string) {
  let result = str.trim();
  if (result.search(/^http[s]?\:\/\//) == -1) {
    result = "http://" + result;
  }
  return result;
}

// Performs inline swap, also handles negative index (counting from the end)
// array_swap([1, 2, 3, 4], 1, 2) => [1, 3, 2, 4]
// array_swap([1, 2, 3, 4], -1, -2) => [1, 2, 4, 3]
export function array_swap(arr: any[], old_index: number, new_index: number) {
  while (old_index < 0) {
    old_index += arr.length;
  }
  while (new_index < 0) {
    new_index += arr.length;
  }
  if (new_index >= arr.length) {
    var k = new_index - arr.length + 1;
    while (k--) {
      arr.push(undefined);
    }
  }
  arr.splice(new_index, 0, arr.splice(old_index, 1)[0]);
}
