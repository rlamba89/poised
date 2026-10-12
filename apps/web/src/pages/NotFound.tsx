import { Center, Stack, Text, Title } from "@mantine/core";

// Any URL that isn't a page (it was Next.js's built-in 404 before plan F2).
export function NotFound() {
  return (
    <Center h="100vh">
      <Stack align="center" gap="xs">
        <Title order={2}>404</Title>
        <Text>This page could not be found.</Text>
      </Stack>
    </Center>
  );
}
