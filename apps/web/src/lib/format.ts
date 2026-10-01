const dateTime = new Intl.DateTimeFormat("en-GB", { dateStyle: "short", timeStyle: "short" });

/** dd/MM/yyyy, HH:mm */
export const formatDateTime = (iso: string) => dateTime.format(new Date(iso));

const date = new Intl.DateTimeFormat("en-GB", { dateStyle: "long", timeZone: "UTC" });

/** "11 December 1974", for a YYYY-MM-DD date such as a date of birth. */
export const formatDate = (ymd: string) => date.format(new Date(ymd + "T00:00:00Z"));

