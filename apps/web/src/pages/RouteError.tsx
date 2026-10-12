import { useEffect, useState } from "react";
import { Link, useRouteError } from "react-router";
import { Anchor, Button, Center, Loader, Stack, Text, Title } from "@mantine/core";
import { isChunkLoadError, reloadOnce } from "@/lib/chunkError";

function sessionStore(): Storage | null {
  try {
    return window.sessionStorage;
  } catch {
    return null;
  }
}

// Shown when a page fails to load or crashes. If the page's code couldn't be downloaded (usually
// the app was updated while open), it reloads once, as Next.js did; otherwise it offers Reload.
export function RouteError() {
  const error = useRouteError();
  const chunk = isChunkLoadError(error);
  const [reloading, setReloading] = useState(chunk);
  useEffect(() => {
    if (chunk && !reloadOnce(sessionStore(), () => window.location.reload(), Date.now())) setReloading(false);
  }, [chunk]);

  if (reloading) {
    return (
      <Center h="100vh">
        <Loader />
      </Center>
    );
  }
  return (
    <Center h="100vh" p="md">
      <Stack align="center" gap="sm">
        <Title order={2}>Something went wrong</Title>
        <Text ta="center">
          {chunk ? "This page couldn't be loaded. Check your connection, then reload." : "Please reload the page."}
        </Text>
        <Button onClick={() => window.location.reload()}>Reload</Button>
        <Anchor component={Link} to="/" size="sm">
          Go to the start page
        </Anchor>
      </Stack>
    </Center>
  );
}
