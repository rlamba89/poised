import { Outlet, ScrollRestoration } from "react-router";

// Every page: a new page starts at the top, and Back returns to where you were (as under Next.js).
export function Root() {
  return (
    <>
      <Outlet />
      <ScrollRestoration />
    </>
  );
}
