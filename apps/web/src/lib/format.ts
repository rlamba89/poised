const dateTime = new Intl.DateTimeFormat("en-GB", { dateStyle: "short", timeStyle: "short" });

/** dd/MM/yyyy, HH:mm */
export const formatDateTime = (iso: string) => dateTime.format(new Date(iso));
