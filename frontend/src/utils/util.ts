import moment from "moment";

export function humanize(ts: number) {
  const time = moment.utc(ts);
  if (moment().year() == time.year()) {
    if (moment().dayOfYear() == time.dayOfYear()) {
      return time.format("HH:mm");
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
