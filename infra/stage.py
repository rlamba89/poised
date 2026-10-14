"""One environment of Poised: ApiStack + WebStack + AuthStack (plan F4's QaStage; AuthStack S12), built from an entry in
countries.py. S07's pipeline deploys these stages; `make deploy-stacks` deploys one by hand."""

from __future__ import annotations

import aws_cdk as cdk
from constructs import Construct

from api_stack import ApiStack
from auth_stack import AuthStack
from budget import COST_TAG
from countries import Environment
from web_stack import WebStack

# Poised's own CDK bootstrap (make aws-setup), apart from any other project's in the account.
QUALIFIER = "poised"
TOOLKIT_STACK = "poised-cdk-toolkit"


class PoisedStage(cdk.Stage):
    def __init__(self, scope: Construct, env: Environment, *, lambda_code: str) -> None:
        super().__init__(scope, env.name, env=cdk.Environment(account=env.account, region=env.region))
        own = dict(synthesizer=cdk.DefaultStackSynthesizer(qualifier=QUALIFIER))
        self.api = ApiStack(
            self, "api", env=env, lambda_code=lambda_code, stack_name=f"poised-{env.name}-api", **own
        )
        self.web = WebStack(
            self, "web", env=env, api_url=self.api.http_api.api_endpoint, stack_name=f"poised-{env.name}-web", **own
        )
        # Last: the app client's callback needs the web address, and staff-api's role gets its
        # invite permission from here, so ApiStack never refers to it (no cycle).
        self.auth = AuthStack(
            self,
            "auth",
            env=env,
            app_url=self.web.app_url,
            staff_api_role=self.api.functions["staff-api"].role,
            stack_name=f"poised-{env.name}-auth",
            **own,
        )
        # Every resource carries its environment, so the budget can count it, and a project tag,
        # so it's clear in a shared account what belongs to Poised.
        cdk.Tags.of(self).add(COST_TAG, env.name)
        cdk.Tags.of(self).add("project", "poised")
