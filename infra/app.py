"""The CDK app: the pipeline (pipeline_stack.py), which holds one PoisedStage (ApiStack +
WebStack) per environment in countries.py and deploys them on a push.

The stages can also be deployed by hand, e.g. `make deploy-stacks ENV=uk-qa` (README.md). The
Lambdas' code is `make build-lambdas`'s output."""

import os

import aws_cdk as cdk

import countries
from pipeline_stack import PipelineStack

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

app = cdk.App()
PipelineStack(
    app,
    "pipeline",
    home=countries.PIPELINE,
    environments=countries.ENVIRONMENTS,
    lambda_code=os.path.join(REPO, "apps", "api", "dist"),
)
app.synth()
