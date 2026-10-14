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


class WebStack(cdk.Stack):
    def __init__(self, scope: Construct, id: str, *, env: Environment, api_url: str, **kwargs) -> None:
        super().__init__(scope, id, **kwargs)
        app = amplify.CfnApp(
            self,
            "App",
            name=f"poised-{env.name}",
            platform="WEB",
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
        ssm.StringParameter(
            self,
            "AppUrlParam",
            parameter_name=f"{env.param_prefix}/app-url",
            string_value=app_url,
            description=f"Poised {env.name}: the web address, for links in texts and emails",
        )

        cdk.CfnOutput(self, "AppId", value=app.attr_app_id)
        cdk.CfnOutput(self, "AppUrl", value=app_url)
