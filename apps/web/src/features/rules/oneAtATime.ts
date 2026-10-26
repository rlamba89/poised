// One save at a time (S40 test report 3, F-1). The Rules page and Test the whole HQ save the whole
// rule set with the revision they loaded, so a second save sent before the first lands would carry
// the same revision and get a false "Someone else changed these rules", losing that change. While a
// save is in flight, another is refused with WAIT_FOR_SAVE (the buttons are disabled too; this holds
// even for a click in the same moment).
export const WAIT_FOR_SAVE = "Wait until the last change has saved, then try again.";

export function oneAtATime<A>(send: (arg: A) => Promise<string>) {
  let inFlight = false;
  return {
    busy: () => inFlight,
    run: async (arg: A): Promise<string> => {
      if (inFlight) return WAIT_FOR_SAVE;
      inFlight = true;
      try {
        return await send(arg);
      } finally {
        inFlight = false;
      }
    },
  };
}
