import "@mantine/core/styles.css";
import "./globals.css";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { createBrowserRouter, RouterProvider } from "react-router";
import { MantineProvider } from "@mantine/core";
import { routes } from "./routes";
import { loadConfig } from "./lib/config";
import { reloadWhenRestored } from "./lib/restore";
import { plainNumberBoxes } from "./lib/numberBoxes";
import { theme } from "./theme";

// Back to a page kept in the browser's back/forward cache reloads it, so a signed-out person's
// screen is never shown again from memory (S12 test report 1, finding 1).
reloadWhenRestored(window);

// A number answer is a plain box: no mouse wheel or arrow keys stepping it (S49a).
plainNumberBoxes(document);

// The router starts loading the first page's code now, while the config is fetched.
const router = createBrowserRouter(routes);

function render() {
  createRoot(document.getElementById("root")!).render(
    <StrictMode>
      <MantineProvider theme={theme}>
        <RouterProvider router={router} />
      </MantineProvider>
    </StrictMode>,
  );
}

// The deployment's locale (GET /api/config) is known before any date is shown. If the API
// can't be reached the pages show their own errors; dates fall back to the browser's locale.
loadConfig()
  .catch((e: unknown) => console.error("GET /api/config failed; dates use the browser's locale.", e))
  .finally(render);
