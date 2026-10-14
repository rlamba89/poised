// /auth/callback (S12): Cognito's managed login sends the browser back here with a code. The page
// checks it's the sign-in this tab started, posts the code and its PKCE verifier to the API (which
// sets our session cookie), then goes where the user starts. A refusal (e.g. a suspended trust, or
// no Poised account) signs out of the managed login too and shows the API's message on /login.
import { useEffect } from "react";
import { useNavigate } from "react-router";
import { Center, Loader, Stack, Text } from "@mantine/core";
import { api } from "@/lib/api";
import { startPath, type Me } from "@/lib/auth";
import { staffSignIn } from "@/lib/config";
import { finishSignIn, refuseSignIn } from "@/lib/signin";

// The page's first load runs its effect twice in development (React's StrictMode), and a code
// works once, so the same sign-in is finished once and both runs wait for it.
const finishing = new Map<string, Promise<string>>();

function finish(search: string): Promise<string> {
  let p = finishing.get(search);
  if (!p) {
    p = (async () => {
      const got = finishSignIn(search);
      if ("error" in got) throw new Error(got.error);
      await api("/auth/callback", { method: "POST", body: JSON.stringify(got) });
      return startPath(await api<Me>("/me"));
    })();
    finishing.set(search, p);
  }
  return p;
}

export function AuthCallback() {
  const navigate = useNavigate();
  useEffect(() => {
    finish(window.location.search)
      .then((to) => navigate(to, { replace: true }))
      .catch((e: Error) => {
        // Also signs out of the managed login, which comes back to /login (finding 2, test report 1).
        const cognito = staffSignIn().cognito;
        refuseSignIn(e.message, cognito);
        if (!cognito) navigate("/login", { replace: true });
      });
  }, [navigate]);
  return (
    <Center h="100vh">
      <Stack align="center">
        <Loader />
        <Text c="dimmed">Signing you in…</Text>
      </Stack>
    </Center>
  );
}
