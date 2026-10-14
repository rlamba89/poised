"""The AWS make targets take PROFILE=<AWS profile>, and it wins over AWS_PROFILE from .env or the
shell (Rahul's .env names another project's profile). Checked with `make -n`, which prints the
commands without running them, so no AWS account is needed."""

import os
import pathlib
import subprocess

import pytest

REPO = pathlib.Path(__file__).resolve().parents[2]

# Each target and the commands in it that talk to AWS.
TARGETS = {
    "aws-setup": ["infra/scripts/aws-setup.sh"],
    "deploy-stacks": ["infra/scripts/check-account.sh", "aws-cdk@"],
    "aws-db": ["infra/scripts/db.sh"],
    "deploy-web": ["infra/scripts/deploy-web.sh"],
    "deploy-qa": [
        "infra/scripts/check-account.sh",
        "aws-cdk@",
        "infra/scripts/db.sh",
        "infra/scripts/qa-users.sh",
        "infra/scripts/deploy-web.sh",
    ],
    "qa-users": ["infra/scripts/qa-users.sh"],
    "qa-code": ["infra/scripts/qa-users.sh"],
    "qa-password": ["infra/scripts/qa-users.sh"],
    "e2e-users": ["infra/scripts/e2e-users.sh"],
    "aws-platform-admin": ["infra/scripts/platform-admin.sh"],
    "destroy-stacks": ["infra/scripts/check-account.sh", "aws-cdk@"],
    "deploy-pipeline": ["infra/scripts/check-account.sh", "aws-cdk@"],
    "destroy-pipeline": ["infra/scripts/check-account.sh", "aws-cdk@"],
}


def dry_run(target: str, *args: str) -> list[str]:
    env = {**os.environ, "AWS_PROFILE": "from-the-shell"}
    out = subprocess.run(
        ["make", "-n", target, *args], cwd=REPO, env=env, capture_output=True, text=True, check=True
    ).stdout
    return out.splitlines()


@pytest.mark.parametrize("target", TARGETS)
def test_profile_is_passed_to_every_aws_command(target):
    lines = dry_run(target, "PROFILE=the-named-profile")
    for marker in TARGETS[target]:
        hits = [line for line in lines if marker in line]
        assert hits, f"make {target}: no command with {marker}"
        for line in hits:
            assert "AWS_PROFILE=the-named-profile" in line, f"make {target}: {line}"


def test_without_profile_nothing_is_forced():
    # Then the shell's (or .env's) AWS_PROFILE applies, as before.
    for line in dry_run("aws-setup"):
        assert "AWS_PROFILE=" not in line, line


def test_hand_deploys_name_the_environments_stage_inside_the_pipeline():
    # The stages live in the pipeline stack (app.py), so their stacks are pipeline/<env>/….
    for target in ("deploy-stacks", "destroy-stacks"):
        cdk = [line for line in dry_run(target, "ENV=xx-qa") if "aws-cdk@" in line]
        assert cdk and all("'pipeline/xx-qa/*'" in line for line in cdk), cdk


def test_the_pipeline_can_skip_docker_compose():
    # It starts its own Postgres, then runs `make test-integration DB_UP=`.
    assert not any("docker compose" in line for line in dry_run("db-up", "DB_UP="))
    assert any("docker compose up" in line for line in dry_run("db-up"))
