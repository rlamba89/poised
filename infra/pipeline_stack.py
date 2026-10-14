"""PipelineStack: CDK Pipelines (plan F4 Step 2). A push to the branch named in countries.py runs
every test, then deploys each environment in countries.py, then, for each, migrates and seeds its
database, deploys the web app and runs the journeys against it.

The pipeline updates itself (CDK Pipelines' "self-mutation"), so after its first deploy
(`make deploy-pipeline`) a change to it is just a push. See README.md."""

from __future__ import annotations

from typing import Sequence

import aws_cdk as cdk
from aws_cdk import aws_codebuild as codebuild
from aws_cdk import aws_codeconnections as codeconnections
from aws_cdk import aws_codepipeline as codepipeline
from aws_cdk import aws_iam as iam
from aws_cdk import aws_s3 as s3
from aws_cdk import pipelines
from constructs import Construct

from countries import Environment, PipelineHome
from stage import QUALIFIER, PoisedStage

# The CDK CLI, as in the Makefile (a test checks they match).
AWS_CDK_VERSION = "2.1145.0"

# The repo's pinned runtimes (CLAUDE.md: Go 1.26+, Node 22.22+; e2e and infra need Python 3.12+).
RUNTIMES = {"golang": "1.26", "nodejs": "22", "python": "3.12"}

# The integration tests' Postgres: the same major version as docker-compose.yml, from ECR Public,
# because Docker Hub limits anonymous pulls and build hosts share addresses.
POSTGRES_IMAGE = "public.ecr.aws/docker/library/postgres:17"

# The synth step, in order. Any failing command stops the run before anything is deployed.
# PYTHON=python3: the build image's Python is the 3.12 named in RUNTIMES.
SYNTH_COMMANDS = [
    "make lint test PYTHON=python3",
    f"docker run -d --name ci-db -e POSTGRES_USER=sj -e POSTGRES_PASSWORD=sj -e POSTGRES_DB=sj -p 5432:5432 {POSTGRES_IMAGE}",
    "for i in $(seq 60); do docker exec ci-db pg_isready -h 127.0.0.1 -U sj && break; sleep 1; done",
    # DB_UP= skips the Makefile's `docker compose up`: the Postgres above is already running.
    "make test-integration DB_UP=",
    "npm run build -w apps/web",
    # make synth builds the Lambdas (make build-lambdas), then `cdk synth` into infra/cdk.out.
    "make synth PYTHON=python3",
]

# Where the journeys' results go: the JUnit report, plus a trace of each journey that failed.
RESULTS_DAYS = 30

ARM = codebuild.BuildEnvironment(
    build_image=codebuild.LinuxArmBuildImage.AMAZON_LINUX_2023_STANDARD_3_0,
    compute_type=codebuild.ComputeType.LARGE,
)


def _spec(runtimes: dict[str, str], post_build: Sequence[str] = ()) -> codebuild.BuildSpec:
    phases: dict = {"install": {"runtime-versions": runtimes}}
    if post_build:
        phases["post_build"] = {"commands": list(post_build)}
    return codebuild.BuildSpec.from_object({"phases": phases})


class PipelineStack(cdk.Stack):
    def __init__(
        self,
        scope: Construct,
        id: str,
        *,
        home: PipelineHome,
        environments: Sequence[Environment],
        lambda_code: str,
        **kwargs,
    ) -> None:
        super().__init__(
            scope,
            id,
            stack_name="poised-pipeline",
            env=cdk.Environment(account=home.account, region=home.region),
            synthesizer=cdk.DefaultStackSynthesizer(qualifier=QUALIFIER),
            **kwargs,
        )
        cdk.Tags.of(self).add("project", "poised")

        # GitHub, through a CodeConnections connection. CloudFormation creates it "Pending"; it
        # works once someone approves it in the console (README.md, the one manual step).
        self.connection = codeconnections.CfnConnection(
            self, "GitHub", connection_name="poised-github", provider_type="GitHub"
        )
        source = pipelines.CodePipelineSource.connection(
            home.repo, home.branch, connection_arn=self.connection.attr_connection_arn
        )

        synth = pipelines.CodeBuildStep(
            "Synth",
            input=source,
            commands=SYNTH_COMMANDS,
            primary_output_directory="infra/cdk.out",
            build_environment=codebuild.BuildEnvironment(
                build_image=ARM.build_image, compute_type=ARM.compute_type, privileged=True
            ),
            partial_build_spec=_spec(RUNTIMES),
            timeout=cdk.Duration.minutes(45),
        )
        web_dist = synth.add_output_directory("apps/web/dist")

        # Deleted with the stack, so an account can be emptied with `cdk destroy` alone.
        artifacts = s3.Bucket(
            self,
            "Artifacts",
            block_public_access=s3.BlockPublicAccess.BLOCK_ALL,
            enforce_ssl=True,
            removal_policy=cdk.RemovalPolicy.DESTROY,
            auto_delete_objects=True,
            lifecycle_rules=[s3.LifecycleRule(expiration=cdk.Duration.days(RESULTS_DAYS))],
        )
        self.results = s3.Bucket(
            self,
            "E2eResults",
            block_public_access=s3.BlockPublicAccess.BLOCK_ALL,
            enforce_ssl=True,
            removal_policy=cdk.RemovalPolicy.DESTROY,
            auto_delete_objects=True,
            lifecycle_rules=[s3.LifecycleRule(expiration=cdk.Duration.days(RESULTS_DAYS))],
        )

        self.pipeline = pipelines.CodePipeline(
            self,
            "Pipeline",
            pipeline_name="poised",
            pipeline_type=codepipeline.PipelineType.V2,  # billed per action minute: ~£0 when idle
            synth=synth,
            artifact_bucket=artifacts,
            cross_account_keys=False,  # one account: no KMS key ($1 a month)
            cli_version=AWS_CDK_VERSION,
            publish_assets_in_parallel=False,  # one asset job, not one per Lambda
        )

        for env in environments:
            stage = PoisedStage(self, env, lambda_code=lambda_code)
            db_web = self._database_and_web(env, source, web_dist)
            e2e = self._e2e(env, source, stage)
            e2e.add_step_dependency(db_web)
            self.pipeline.add_stage(stage, post=[db_web, e2e])

        self.pipeline.build_pipeline()
        cdk.CfnOutput(self, "ConnectionArn", value=self.connection.attr_connection_arn)
        cdk.CfnOutput(self, "E2eResultsBucket", value=self.results.bucket_name)

    def _database_and_web(
        self, env: Environment, source: pipelines.IFileSetProducer, web_dist: pipelines.FileSet
    ) -> pipelines.CodeBuildStep:
        """Migrate + seed the environment's database (infra/scripts/db.sh), then upload the web
        app that the synth step built (infra/scripts/deploy-web.sh): what `make aws-db` and
        `make deploy-web` do by hand."""
        arn = lambda service, resource: f"arn:aws:{service}:{env.region}:{env.account}:{resource}"  # noqa: E731
        commands = [f"bash infra/scripts/db.sh {env.name}"]
        test_users: list[iam.PolicyStatement] = []
        if env.stage == "qa":
            # QA only (S12): the seed's staff get sign-ins with known passwords and authenticator
            # secrets (cmd/qa-users), kept in one SecureString, for the tester and the journeys.
            commands.append(f"bash infra/scripts/qa-users.sh {env.name} setup")
            test_users = [
                iam.PolicyStatement(
                    actions=["ssm:GetParameter"],
                    resources=[
                        arn("ssm", f"parameter{env.param_prefix}/{p}")
                        for p in ("test-users", "cognito-user-pool-id", "cognito-client-id")
                    ],
                ),
                iam.PolicyStatement(
                    actions=["ssm:PutParameter"], resources=[arn("ssm", f"parameter{env.param_prefix}/test-users")]
                ),
                # The pool's ARN isn't known here (it's in the stage's AuthStack), so the actions
                # are limited to the pool tagged as this environment's: the account has another
                # project's pools.
                iam.PolicyStatement(
                    actions=[
                        "cognito-idp:AdminGetUser",
                        "cognito-idp:AdminCreateUser",
                        "cognito-idp:AdminDeleteUser",
                        "cognito-idp:AdminSetUserPassword",
                        "cognito-idp:AdminInitiateAuth",
                        "cognito-idp:AdminRespondToAuthChallenge",
                        "cognito-idp:AdminSetUserMFAPreference",
                    ],
                    resources=[arn("cognito-idp", "userpool/*")],
                    conditions={"StringEquals": {"aws:ResourceTag/poised-env": env.name}},
                ),
            ]
        commands.append(f"bash infra/scripts/deploy-web.sh {env.name}")
        return pipelines.CodeBuildStep(
            "Database-and-web",
            input=source,
            additional_inputs={"apps/web/dist": web_dist},
            commands=commands,
            build_environment=ARM,
            partial_build_spec=_spec({"golang": RUNTIMES["golang"], "python": RUNTIMES["python"]}),
            role_policy_statements=[
                *test_users,
                iam.PolicyStatement(
                    actions=["ssm:GetParameter"],
                    resources=[arn("ssm", f"parameter{env.param_prefix}/database-url")],
                ),
                iam.PolicyStatement(
                    actions=["cloudformation:DescribeStacks"],
                    resources=[arn("cloudformation", f"stack/poised-{env.name}-web/*")],
                ),
                iam.PolicyStatement(
                    actions=["amplify:CreateDeployment", "amplify:StartDeployment", "amplify:GetJob"],
                    resources=[arn("amplify", "apps/*")],
                ),
            ],
        )

    def _e2e(self, env: Environment, source: pipelines.IFileSetProducer, stage: PoisedStage) -> pipelines.CodeBuildStep:
        """The journeys (make e2e) against the environment's web address. Ubuntu, because
        Playwright's WebKit (the patient's iPhone) supports only Ubuntu and Debian. The results go
        to the E2eResults bucket after the run, whether it passed or failed."""
        where = f"s3://$RESULTS_BUCKET/{env.name}/$CODEBUILD_BUILD_NUMBER/"
        users = f"arn:aws:ssm:{env.region}:{env.account}:parameter{env.param_prefix}/test-users"
        qa = env.stage == "qa"  # only QA has test users (S12); no other stage exists yet (pending.md)
        return pipelines.CodeBuildStep(
            "E2E",
            input=source,
            env={"RESULTS_BUCKET": self.results.bucket_name},
            env_from_cfn_outputs={"E2E_BASE_URL": stage.web.app_url_output},
            commands=[
                "make e2e-setup PYTHON=python3",
                "e2e/.venv/bin/playwright install-deps chromium webkit",
                # The test users' sign-ins (S12), into a file the journeys read; never printed.
                *([f"bash infra/scripts/e2e-users.sh {env.name}"] if qa else []),
                # E2E_ARGS: extra pytest options for one run, e.g. --tracing=on (README.md).
                'make e2e ARGS="--junitxml=test-results/junit.xml $E2E_ARGS"',
            ],
            build_environment=codebuild.BuildEnvironment(
                build_image=codebuild.LinuxBuildImage.from_code_build_image_id("aws/codebuild/standard:8.0"),
                compute_type=codebuild.ComputeType.MEDIUM,
            ),
            partial_build_spec=_spec(
                {"python": RUNTIMES["python"]},
                post_build=[
                    f'aws s3 cp --recursive --only-show-errors e2e/test-results "{where}" || true',
                    f'echo "The journeys\' JUnit report, and a trace of each that failed: {where}"',
                ],
            ),
            role_policy_statements=[
                iam.PolicyStatement(actions=["s3:PutObject"], resources=[self.results.arn_for_objects(f"{env.name}/*")]),
                *([iam.PolicyStatement(actions=["ssm:GetParameter"], resources=[users])] if qa else []),
            ],
        )
