"""ApiStack: the Go API on Lambda behind an API Gateway HTTP API (plan F4 Step 1), for one
environment in countries.py. Nothing here names a region, a domain or a country default: they
all come from the entry (A-20)."""

from __future__ import annotations

import os

import aws_cdk as cdk
from aws_cdk import aws_apigatewayv2 as apigw
from aws_cdk import aws_apigatewayv2_integrations as integrations
from aws_cdk import aws_iam as iam
from aws_cdk import aws_lambda as lambda_
from aws_cdk import aws_logs as logs
from aws_cdk import aws_secretsmanager as secretsmanager
from constructs import Construct

from budget import Budget
from countries import Environment

# The functions built by `make build-lambdas` (apps/api/dist/<fn>/bootstrap).
FUNCTIONS = ("staff-api", "patient-api")

# Which address in X-Forwarded-For is the browser's, counting from the right (config
# FORWARDED_HOPS), for the patient sign-in rate limit. Measured on QA (S06, 14 Oct) through
# Amplify's rewrite: "<browser>, <CloudFront edge>, <Amplify's outbound address>", the last added
# by API Gateway; GET /api/qa/client shows it.
FORWARDED_HOPS = 3


class ApiStack(cdk.Stack):
    def __init__(self, scope: Construct, id: str, *, env: Environment, lambda_code: str, **kwargs) -> None:
        super().__init__(scope, id, **kwargs)
        self.environment_entry = env

        # The session cookies' signing key: generated here, so no one ever sees or types it.
        # Deleted with the stack (a retained secret would cost money and block a re-create).
        self.token_secret = secretsmanager.Secret(
            self,
            "TokenSecret",
            description=f"Poised {env.name}: signs the session cookies",
            generate_secret_string=secretsmanager.SecretStringGenerator(
                password_length=64, exclude_punctuation=True
            ),
            removal_policy=cdk.RemovalPolicy.DESTROY,
        )

        # The database URL is a SecureString that `make aws-setup` stores (CloudFormation can't
        # create SecureStrings); the web address is written by WebStack. Both are read once, at
        # cold start (internal/config Resolve).
        db_param = f"{env.param_prefix}/database-url"
        app_url_param = f"{env.param_prefix}/app-url"
        variables = {
            "CONFIG_STRICT": "true",  # no local defaults: a missing value stops the function
            "DATABASE_URL_PARAM": db_param,
            "APP_URL_PARAM": app_url_param,
            "TOKEN_SECRET_ARN": self.token_secret.secret_arn,
            "DEFAULT_TIME_ZONE": env.time_zone,
            "DEFAULT_LOCALE": env.locale,
            "PHONE_COUNTRY": env.phone_country,
            "NATIONAL_ID_TYPE": env.national_id_type,
            "MESSAGING": env.messaging,
            "FORWARDED_HOPS": str(FORWARDED_HOPS),
        }
        if env.dev_login:  # countries.py refuses it outside QA
            variables["DEV_LOGIN"] = "true"

        read_params = iam.PolicyStatement(
            actions=["ssm:GetParameter"],
            resources=[
                self.format_arn(service="ssm", resource="parameter", resource_name=p.lstrip("/"))
                for p in (db_param, app_url_param)
            ],
        )

        self.functions: dict[str, lambda_.Function] = {}
        for fn in FUNCTIONS:
            code = os.path.join(lambda_code, fn)
            if not os.path.isfile(os.path.join(code, "bootstrap")):
                raise FileNotFoundError(f"{code}/bootstrap is missing: run `make build-lambdas` first")
            log_group = logs.LogGroup(
                self,
                f"{fn}-logs",
                retention=logs.RetentionDays.TWO_WEEKS,
                removal_policy=cdk.RemovalPolicy.DESTROY,
            )
            f = lambda_.Function(
                self,
                fn,
                function_name=f"poised-{env.name}-{fn}",
                runtime=lambda_.Runtime.PROVIDED_AL2023,
                architecture=lambda_.Architecture.ARM_64,
                handler="bootstrap",
                code=lambda_.Code.from_asset(code),
                memory_size=512,
                # API Gateway gives up after 30 s; a sleeping Neon database wakes in a few.
                timeout=cdk.Duration.seconds(25),
                environment=variables,
                log_group=log_group,
            )
            f.add_to_role_policy(read_params)
            self.token_secret.grant_read(f)
            self.functions[fn] = f

        # /api/p/… is the patient's link (patient-api); every other /api/… is staff-api (D-4).
        self.http_api = apigw.HttpApi(self, "HttpApi", api_name=f"poised-{env.name}")
        self.http_api.add_routes(
            path="/api/p/{proxy+}",
            methods=[apigw.HttpMethod.ANY],
            integration=integrations.HttpLambdaIntegration("patient-api", self.functions["patient-api"]),
        )
        self.http_api.add_routes(
            path="/api/{proxy+}",
            methods=[apigw.HttpMethod.ANY],
            integration=integrations.HttpLambdaIntegration("staff-api", self.functions["staff-api"]),
        )

        Budget(self, "Budget", env=env)

        cdk.CfnOutput(self, "ApiUrl", value=self.http_api.api_endpoint)
        cdk.CfnOutput(self, "TokenSecretArn", value=self.token_secret.secret_arn)
