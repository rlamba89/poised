"""ApiStack: the Go API on Lambda behind an API Gateway HTTP API (plan F4 Step 1), for one
environment in countries.py. Nothing here names a region, a domain or a country default: they
all come from the entry (A-20)."""

from __future__ import annotations

import os

import aws_cdk as cdk
from aws_cdk import aws_apigatewayv2 as apigw
from aws_cdk import aws_apigatewayv2_integrations as integrations
from aws_cdk import aws_cloudwatch as cloudwatch
from aws_cdk import aws_cloudwatch_actions as cw_actions
from aws_cdk import aws_iam as iam
from aws_cdk import aws_lambda as lambda_
from aws_cdk import aws_logs as logs
from aws_cdk import aws_scheduler as scheduler
from aws_cdk import aws_scheduler_targets as targets
from aws_cdk import aws_secretsmanager as secretsmanager
from aws_cdk import aws_sns as sns
from aws_cdk import aws_sns_subscriptions as subscriptions
from aws_cdk import aws_ssm as ssm
from constructs import Construct

from budget import ALERT_EMAIL_PARAM, Budget
from countries import Environment
from messaging_stack import configuration_set_name, email_from_param, test_recipients_param

# The functions built by `make build-lambdas` (apps/api/dist/<fn>/bootstrap): the two behind the
# HTTP API, and the jobs Lambda (S17), run by its schedule and by the delivery topic.
FUNCTIONS = ("staff-api", "patient-api", "jobs")

# Every log line about a text or email that couldn't be sent or wasn't delivered starts with this
# (Go's messaging.DeliveryFailedMarker); the alarm counts them (OPS-05).
DELIVERY_FAILED_MARKER = "DELIVERY_FAILED"

# The staff user pool's values the functions read from SSM (config.Resolve): variable → the
# parameter's last part, <prefix>/cognito-<part>. AuthStack writes them.
COGNITO_PARAMS = {
    "STAFF_POOL_ID": "user-pool-id",
    "STAFF_POOL_CLIENT_ID": "client-id",
    "STAFF_POOL_DOMAIN": "domain",
}

# Which address in X-Forwarded-For is the browser's, counting from the right (config
# FORWARDED_HOPS), for the patient sign-in rate limit. Measured on QA (S06, 14 Oct) through
# Amplify's rewrite: "<browser>, <CloudFront edge>, <Amplify's outbound address>", the last added
# by API Gateway; GET /api/qa/client shows it.
FORWARDED_HOPS = 3


class ApiStack(cdk.Stack):
    def __init__(
        self,
        scope: Construct,
        id: str,
        *,
        env: Environment,
        lambda_code: str,
        deliveries: sns.ITopic | None = None,
        **kwargs,
    ) -> None:
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
        # The staff user pool (S12), written by AuthStack, which comes after WebStack (its
        # callback addresses need the web address), so these are read at cold start too.
        cognito_params = {name: f"{env.param_prefix}/cognito-{suffix}" for name, suffix in COGNITO_PARAMS.items()}
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
            # The jobs Lambda's schedule (S17), for the testers' jobs page; no API timer on AWS.
            "JOBS_SCHEDULE_MINUTES": str(env.jobs_every_minutes),
            "JOBS_TIMER": "false",
            # No DEV_LOGIN on AWS (S12): staff sign in with Cognito, and Go refuses it when strict.
            **{f"{name}_PARAM": param for name, param in cognito_params.items()},
        }

        params = [db_param, app_url_param, *cognito_params.values()]
        # Real sending (S17): the SES sender's address is in SSM (not the repo), and QA's test
        # recipients are a SecureString there (none: QA sends to no one).
        if env.sends:
            variables |= {
                "EMAIL_FROM_PARAM": email_from_param(env),
                "SES_CONFIGURATION_SET": configuration_set_name(env),
                "SMS_CONFIGURATION_SET": configuration_set_name(env),
            }
            params.append(email_from_param(env))
            if env.sms_sender_id:
                variables["SMS_SENDER_ID"] = env.sms_sender_id
        if env.qa_test_recipients:
            variables["MESSAGING_TEST_RECIPIENTS_PARAM"] = test_recipients_param(env)
            params.append(test_recipients_param(env))
        read_params = iam.PolicyStatement(
            actions=["ssm:GetParameter"],
            resources=[self.format_arn(service="ssm", resource="parameter", resource_name=p.lstrip("/")) for p in params],
        )
        send = [
            iam.PolicyStatement(
                actions=["ses:SendEmail"],
                resources=[
                    self.format_arn(service="ses", resource="identity", resource_name="*"),
                    self.format_arn(service="ses", resource="configuration-set", resource_name=configuration_set_name(env)),
                ],
            ),
            iam.PolicyStatement(
                actions=["sms-voice:SendTextMessage"],
                resources=[self.format_arn(service="sms-voice", resource="*")],
            ),
        ]

        self.functions: dict[str, lambda_.Function] = {}
        self.log_groups: dict[str, logs.LogGroup] = {}
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
                # API Gateway gives up after 30 s; a sleeping Neon database wakes in a few. The
                # jobs run in batches (at most 500 a run) on a schedule with no one waiting.
                timeout=cdk.Duration.minutes(3) if fn == "jobs" else cdk.Duration.seconds(25),
                environment=variables,
                log_group=log_group,
            )
            f.add_to_role_policy(read_params)
            if env.sends:
                for statement in send:
                    f.add_to_role_policy(statement)
            self.token_secret.grant_read(f)
            self.functions[fn] = f
            self.log_groups[fn] = log_group

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

        jobs = self.functions["jobs"]
        # The jobs (S16's jobs.Runner) on a schedule (S17). A failed run isn't retried: the next
        # comes in minutes, and the jobs themselves retry what failed.
        scheduler.Schedule(
            self,
            "JobsSchedule",
            schedule_name=f"poised-{env.name}-jobs",
            description=f"Poised {env.name}: reminders and 'not responding'",
            schedule=scheduler.ScheduleExpression.rate(cdk.Duration.minutes(env.jobs_every_minutes)),
            target=targets.LambdaInvoke(
                jobs, input=scheduler.ScheduleTargetInput.from_object({"run": "jobs"}), retry_attempts=0
            ),
        )
        if deliveries is not None:  # MessagingStack's topic: undelivered texts and emails (OPS-05)
            deliveries.add_subscription(subscriptions.LambdaSubscription(jobs))

        self._alarms(env)
        Budget(self, "Budget", env=env)

        cdk.CfnOutput(self, "ApiUrl", value=self.http_api.api_endpoint)

    def _alarms(self, env: Environment) -> None:
        """OPS-05: an email when a text or email isn't delivered, or a jobs run fails. The address
        is the budget alarm's (SSM, from `make aws-setup`); AWS asks it to confirm once."""
        topic = sns.Topic(self, "Alarms", topic_name=f"poised-{env.name}-alarms")
        sns.Subscription(
            self,
            "AlarmEmail",
            topic=topic,
            protocol=sns.SubscriptionProtocol.EMAIL,
            endpoint=ssm.StringParameter.value_for_string_parameter(self, ALERT_EMAIL_PARAM),
        )
        action = cw_actions.SnsAction(topic)
        namespace = f"Poised/{env.name}"
        for fn, group in self.log_groups.items():
            logs.MetricFilter(
                self,
                f"{fn}-delivery-failed",
                log_group=group,
                filter_pattern=logs.FilterPattern.literal(f'"{DELIVERY_FAILED_MARKER}"'),
                metric_namespace=namespace,
                metric_name="DeliveryFailures",
                metric_value="1",
            )
        failures = cloudwatch.Alarm(
            self,
            "DeliveryFailures",
            alarm_name=f"poised-{env.name}-delivery-failures",
            alarm_description="A patient's text or email couldn't be sent or wasn't delivered (the logs' DELIVERY_FAILED lines).",
            metric=cloudwatch.Metric(
                namespace=namespace, metric_name="DeliveryFailures", statistic="Sum", period=cdk.Duration.minutes(5)
            ),
            threshold=1,
            evaluation_periods=1,
            comparison_operator=cloudwatch.ComparisonOperator.GREATER_THAN_OR_EQUAL_TO_THRESHOLD,
            treat_missing_data=cloudwatch.TreatMissingData.NOT_BREACHING,
        )
        jobs_failed = cloudwatch.Alarm(
            self,
            "JobsFailed",
            alarm_name=f"poised-{env.name}-jobs-failed",
            alarm_description="A jobs Lambda run failed (the schedule's or a delivery event's).",
            metric=self.functions["jobs"].metric_errors(period=cdk.Duration.minutes(5), statistic="Sum"),
            threshold=1,
            evaluation_periods=1,
            comparison_operator=cloudwatch.ComparisonOperator.GREATER_THAN_OR_EQUAL_TO_THRESHOLD,
            treat_missing_data=cloudwatch.TreatMissingData.NOT_BREACHING,
        )
        for alarm in (failures, jobs_failed):
            alarm.add_alarm_action(action)
        cdk.CfnOutput(self, "TokenSecretArn", value=self.token_secret.secret_arn)
