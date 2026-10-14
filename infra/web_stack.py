"""WebStack: the React app on Amplify Hosting (plan F4 Step 1), deployed by hand from a zip
(no repository connected), with Amplify proxying /api/* to the HTTP API so the browser sees
one origin and the session cookies just work."""

from __future__ import annotations

import aws_cdk as cdk
from aws_cdk import aws_amplify as amplify
from aws_cdk import aws_ssm as ssm
from constructs import Construct

from countries import Environment

BRANCH = "main"

# The SPA rule, in Amplify's regex form: every page's address serves /index.html (React Router
# draws the page), but a file, by its extension, and anything under /assets/ is served as itself,
# so a file that's gone after a deploy is a real 404 and the app's "reload once" can tell
# (pending.md, S04 verification 2). A page's last part may hold a dot (e.g. a token).
SPA_REWRITE_SOURCE = (
    r"</^(?!\/assets\/)([^.]*|.*\.(?!(css|js|mjs|map|json|txt|xml|ico|png|jpg|jpeg|gif|svg|webp"
    r"|woff|woff2|ttf|webmanifest)$)[^.]*)$/>"
)


# Security headers on every page and file (S12; S06 test report 1): no framing (clickjacking),
# https only, no MIME sniffing, and no Referer, so a patient's link never leaves in one (S14).
# The CSP sets only frame-ancestors, so it can't break the app's own scripts and styles.
# Pages are never stored (no-store), so Back after a sign-out on a shared workstation can't show
# the previous person's screen from the browser's cache (S12 test report 1, finding 1). The
# hashed files under /assets/ change name with their content, so they're kept for good instead.
SECURITY_HEADERS = """customHeaders:
  - pattern: '**'
    headers:
      - key: Cache-Control
        value: no-store
      - key: Strict-Transport-Security
        value: max-age=31536000; includeSubDomains
      - key: X-Frame-Options
        value: DENY
      - key: Content-Security-Policy
        value: frame-ancestors 'none'
      - key: X-Content-Type-Options
        value: nosniff
      - key: Referrer-Policy
        value: no-referrer
  - pattern: 'assets/**'
    headers:
      - key: Cache-Control
        value: public, max-age=31536000, immutable
"""


class WebStack(cdk.Stack):
    def __init__(self, scope: Construct, id: str, *, env: Environment, api_url: str, **kwargs) -> None:
        super().__init__(scope, id, **kwargs)
        app = amplify.CfnApp(
            self,
            "App",
            name=f"poised-{env.name}",
            platform="WEB",
            custom_headers=SECURITY_HEADERS,
            # In this order: the API first (a reverse proxy, 200), then the SPA rule.
            custom_rules=[
                amplify.CfnApp.CustomRuleProperty(source="/api/<*>", target=f"{api_url}/api/<*>", status="200"),
                amplify.CfnApp.CustomRuleProperty(source=SPA_REWRITE_SOURCE, target="/index.html", status="200"),
            ],
        )
        amplify.CfnBranch(self, "Branch", app_id=app.attr_app_id, branch_name=BRANCH, enable_auto_build=False)

        # Until there's a domain the address is Amplify's own, known only once the app exists, so
        # the functions read it from here at cold start rather than from their environment.
        app_url = env.app_url or f"https://{BRANCH}.{app.attr_default_domain}"
        self.app_url = app_url  # AuthStack's callback and sign-out addresses
        ssm.StringParameter(
            self,
            "AppUrlParam",
            parameter_name=f"{env.param_prefix}/app-url",
            string_value=app_url,
            description=f"Poised {env.name}: the web address, for links in texts and emails",
        )

        cdk.CfnOutput(self, "AppId", value=app.attr_app_id)
        # The pipeline's journeys run against this address (pipeline_stack.py).
        self.app_url_output = cdk.CfnOutput(self, "AppUrl", value=app_url)
