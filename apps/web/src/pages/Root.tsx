import { useEffect } from "react";
import { Outlet, ScrollRestoration } from "react-router";
import { pageRendered, sessionStore } from "@/lib/chunkError";

// Every page: a new page starts at the top, and Back returns to where you were (as under Next.js).
export function Root() {
  // A page rendered (an error shows RouteError instead), so a later missing file, e.g. after the
  // next deploy, may reload once again.
  useEffect(() => pageRendered(sessionStore()), []);
  return (
    <>
      <Outlet />
      <ScrollRestoration />
    </>
  );
}
