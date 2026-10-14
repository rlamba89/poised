"""The CDK app: one PoisedStage (ApiStack + WebStack) per environment in countries.py.

Deploy one by hand with `make deploy-stacks ENV=uk-qa` (README.md); S07's pipeline will deploy
them on a push. The Lambdas' code is `make build-lambdas`'s output."""

import os

import aws_cdk as cdk

import countries
from stage import PoisedStage

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

app = cdk.App()
for env in countries.ENVIRONMENTS:
    PoisedStage(app, env, lambda_code=os.path.join(REPO, "apps", "api", "dist"))
app.synth()
