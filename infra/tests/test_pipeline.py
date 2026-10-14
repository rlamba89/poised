"""The pipeline (plan F4 Step 2), written first: a push to the branch runs every test, deploys
each environment in countries.py, then migrates, seeds, deploys the web app and runs the journeys
against it. No AWS account is needed."""

import json
import pathlib
import re

import aws_cdk as cdk
import pytest
from aws_cdk.assertions import Template

import countries
from countries import PipelineHome
from pipeline_stack import AWS_CDK_VERSION, SYNTH_COMMANDS, PipelineStack
from test_stacks import CA_QA

HOME = PipelineHome(account="111111111111", region="ca-central-1", repo="someone/poised", branch="trunk")


@pytest.fixture
def lambda_code(tmp_path):
    for fn in ("staff-api", "patient-api"):
        (tmp_path / fn).mkdir()
        (tmp_path / fn / "bootstrap").write_text("not a real binary")
    return tmp_path


@pytest.fixture
def stack(lambda_code):
    app = cdk.App()
    return PipelineStack(app, "pipeline", home=HOME, environments=[CA_QA], lambda_code=str(lambda_code))


@pytest.fixture
def template(stack):
    return Template.from_stack(stack)


def projects(t: Template) -> dict[str, dict]:
    """Every CodeBuild project, by a readable name, with its buildspec parsed."""
    out = {}
    for logical, p in t.find_resources("AWS::CodeBuild::Project").items():
        props = p["Properties"]
        spec = props["Source"].get("BuildSpec", "{}")
        if isinstance(spec, str) and not spec.startswith("{"):
            spec = "{}"  # the asset jobs' buildspec is a file in the cloud assembly
        props["spec"] = json.loads(spec) if isinstance(spec, str) else spec
        out[logical] = props
    return out


def project(t: Template, name_part: str) -> dict:
    hits = {k: v for k, v in projects(t).items() if name_part.lower() in k.lower()}
    assert len(hits) == 1, f"{name_part}: {list(projects(t))}"
    return next(iter(hits.values()))


def commands(p: dict, phase: str = "build") -> list[str]:
    return p["spec"]["phases"].get(phase, {}).get("commands", [])


def actions(t: Template) -> list[tuple[str, dict]]:
    pipeline = next(iter(t.find_resources("AWS::CodePipeline::Pipeline").values()))["Properties"]
    return [(s["Name"], a) for s in pipeline["Stages"] for a in s["Actions"]]


# --- The source: GitHub, through a connection CDK creates ----------------------------------------


def test_the_github_connection_is_created_by_cdk(template):
    template.has_resource_properties(
        "AWS::CodeConnections::Connection", {"ConnectionName": "poised-github", "ProviderType": "GitHub"}
    )


def test_the_source_is_the_repo_and_branch_in_countries_py(template):
    source = [a for stage, a in actions(template) if stage == "Source"]
    assert len(source) == 1
    config = source[0]["Configuration"]
    assert config["FullRepositoryId"] == "someone/poised"
    assert config["BranchName"] == "trunk"
    assert config["ConnectionArn"] == {"Fn::GetAtt": ["GitHub", "ConnectionArn"]}  # the connection in this stack


def test_the_real_pipeline_follows_the_repos_default_branch():
    assert countries.PIPELINE.repo == "rlamba89/poised"
    assert countries.PIPELINE.branch == "master"  # the GitHub repo's default branch (S07)


def test_a_v2_pipeline_with_no_kms_key(template):
    template.has_resource_properties("AWS::CodePipeline::Pipeline", {"PipelineType": "V2", "Name": "poised"})
    template.resource_count_is("AWS::KMS::Key", 0)  # $1 a month each; nothing is cross-account


# --- The synth step: every test before anything is deployed --------------------------------------


def test_synth_runs_every_test_then_builds_then_synthesises(template):
    synth = project(template, "Synth")
    cmds = commands(synth)
    assert cmds == SYNTH_COMMANDS

    def at(fragment: str) -> int:
        for i, c in enumerate(cmds):
            if fragment in c:
                return i
        raise AssertionError(f"no command with {fragment!r}: {cmds}")

    assert at("make lint test") < at("docker run") < at("make test-integration") < at("make synth")
    assert at("npm run build -w apps/web") < at("make synth")
    # The integration tests use the Postgres the step starts, not docker compose.
    assert "DB_UP=" in cmds[at("make test-integration")]
    # Postgres from ECR Public (Docker Hub limits anonymous pulls from shared build hosts), at the
    # same major version as docker-compose.yml.
    compose = (pathlib.Path(__file__).resolve().parents[2] / "docker-compose.yml").read_text()
    major = re.search(r"image: postgres:(\d+)", compose).group(1)
    assert f"public.ecr.aws/docker/library/postgres:{major}" in cmds[at("docker run")]


def test_synth_runs_docker_on_arm_with_the_repos_runtimes(template):
    synth = project(template, "Synth")
    env = synth["Environment"]
    assert env["PrivilegedMode"] is True  # Docker, for the integration tests' Postgres
    assert env["Type"] == "ARM_CONTAINER"  # the Lambdas are arm64
    runtimes = synth["spec"]["phases"]["install"]["runtime-versions"]
    assert runtimes == {"golang": "1.26", "nodejs": "22", "python": "3.12"}


def test_synth_outputs_the_cloud_assembly_and_the_web_app(template):
    synth = project(template, "Synth")
    artifacts = synth["spec"]["artifacts"]
    dirs = {artifacts.get("base-directory")} | {
        a.get("base-directory") for a in artifacts.get("secondary-artifacts", {}).values()
    }
    assert {"infra/cdk.out", "apps/web/dist"} <= dirs


def test_the_pipeline_uses_the_makefiles_cdk_version(template):
    makefile = (pathlib.Path(__file__).resolve().parents[2] / "Makefile").read_text()
    assert f"AWS_CDK_VERSION := {AWS_CDK_VERSION}" in makefile
    # The self-mutation step installs the same CLI.
    assert any(f"aws-cdk@{AWS_CDK_VERSION}" in c for p in projects(template).values() for c in commands(p, "install"))


# --- Each environment: deploy, then migrate + seed + web, then the journeys ---------------------


def stage_actions(t: Template, env_name: str) -> dict[str, dict]:
    return {a["Name"]: a for stage, a in actions(t) if stage == env_name}


def test_each_environment_is_deployed_then_set_up_then_tested(template):
    acts = stage_actions(template, "ca-qa")
    deploys = [n for n in acts if n.endswith(".Deploy")]
    assert len(deploys) == 2  # ApiStack and WebStack
    db_web = acts["Database-and-web"]
    e2e = acts["E2E"]
    assert all(acts[d]["RunOrder"] < db_web["RunOrder"] for d in deploys)
    assert db_web["RunOrder"] < e2e["RunOrder"]


def test_database_and_web_step(template):
    cmds = commands(project(template, "Databaseandweb"))
    assert cmds == ["bash infra/scripts/db.sh ca-qa", "bash infra/scripts/deploy-web.sh ca-qa"]


def test_database_and_web_step_can_only_do_its_job(template):
    p = project(template, "Databaseandweb")
    role = p["ServiceRole"]
    policies = [
        r["Properties"]["PolicyDocument"]["Statement"]
        for r in template.find_resources("AWS::IAM::Policy").values()
        if role["Fn::GetAtt"][0] in json.dumps(r["Properties"]["Roles"])
    ]
    statements = json.dumps(policies)
    assert "ssm:GetParameter" in statements and "/sj/qa/database-url" in statements
    assert "ca-central-1" in statements  # the environment's own region
    assert "amplify:CreateDeployment" in statements and "amplify:StartDeployment" in statements
    assert "poised-ca-qa-web" in statements  # its stack's outputs
    assert "ssm:PutParameter" not in statements


def test_e2e_runs_the_journeys_against_the_environments_web_address(template):
    p = project(template, "E2E")
    cmds = commands(p)
    assert any("make e2e-setup" in c for c in cmds)
    assert any("playwright install-deps" in c for c in cmds)
    assert any(c.startswith("make e2e ") and "--junitxml" in c for c in cmds)
    # E2E_BASE_URL comes from WebStack's AppUrl output, through the pipeline's action.
    action = stage_actions(template, "ca-qa")["E2E"]
    variables = json.loads(action["Configuration"]["EnvironmentVariables"])
    [base_url] = [v for v in variables if v["name"] == "E2E_BASE_URL"]
    assert re.fullmatch(r"#\{pipelinecaqaweb\w+\.AppUrl\}", base_url["value"]), base_url
    # Ubuntu: Playwright's WebKit (the patient's iPhone) needs Ubuntu's libraries.
    assert p["Environment"]["Image"].startswith("aws/codebuild/standard:")


def test_e2e_keeps_its_traces_whether_it_passes_or_fails(template):
    p = project(template, "E2E")
    after = " ".join(commands(p, "post_build"))
    assert "aws s3 cp" in after and "e2e/test-results" in after
    template.has_resource_properties(
        "AWS::S3::Bucket",
        {"LifecycleConfiguration": {"Rules": [{"ExpirationInDays": 30, "Status": "Enabled"}]}},
    )


def test_nothing_of_the_pipelines_home_country_in_another_countrys_steps(lambda_code):
    """A-20: the steps for ca-qa name ca-qa, not the UK's entry."""
    app = cdk.App()
    stack = PipelineStack(app, "pipeline", home=HOME, environments=[CA_QA], lambda_code=str(lambda_code))
    everything = json.dumps(Template.from_stack(stack).to_json())
    for uk in ["uk-qa", "eu-west-2", "Europe/London", "en-GB"]:
        assert uk not in everything


def test_the_pipeline_and_its_buckets_are_deleted_with_the_stack(template):
    for bucket in template.find_resources("AWS::S3::Bucket").values():
        assert bucket["DeletionPolicy"] == "Delete"


def test_the_pipeline_uses_poiseds_own_bootstrap(template):
    assert template.to_json()["Parameters"]["BootstrapVersion"]["Default"] == "/cdk-bootstrap/poised/version"


def test_the_real_pipeline_synthesises(lambda_code):
    app = cdk.App()
    PipelineStack(
        app, "pipeline", home=countries.PIPELINE, environments=countries.ENVIRONMENTS, lambda_code=str(lambda_code)
    )
    app.synth()
