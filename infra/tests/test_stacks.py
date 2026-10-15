"""The CDK app's tests (plan F4 Step 1), written first. They synthesise the stacks for a made-up
second country and check that nothing of the UK leaks into them (A-20), then check the pieces
each stack must have. No AWS account is needed."""

import dataclasses
import json
import re

import aws_cdk as cdk
import pytest
from aws_cdk.assertions import Match, Template

import countries
from countries import Environment
from stage import PoisedStage
from web_stack import SPA_REWRITE_SOURCE

# A made-up second country, in another region and account.
CA_QA = Environment(
    name="ca-qa",
    stage="qa",
    account="111111111111",
    region="ca-central-1",
    time_zone="America/Toronto",
    locale="en-CA",
    phone_country="CA",
    national_id_type="nhs",
)

UK_VALUES = ["eu-west-2", "Europe/London", "en-GB", "London", '"GB"']


@pytest.fixture
def lambda_code(tmp_path):
    """Stands in for `make build-lambdas`: apps/api/dist/<fn>/bootstrap."""
    for fn in ("staff-api", "patient-api", "jobs"):
        (tmp_path / fn).mkdir()
        (tmp_path / fn / "bootstrap").write_text("not a real binary")
    return tmp_path


def synth(env: Environment, lambda_code) -> dict[str, Template]:
    app = cdk.App()
    stage = PoisedStage(app, env, lambda_code=str(lambda_code))
    return {
        "api": Template.from_stack(stage.api),
        "web": Template.from_stack(stage.web),
        "auth": Template.from_stack(stage.auth),
    }


def text(templates: dict[str, Template]) -> str:
    return json.dumps([t.to_json() for t in templates.values()])


def test_a_second_country_has_nothing_of_the_uk(lambda_code):
    templates = synth(CA_QA, lambda_code)
    everything = text(templates)
    for uk in UK_VALUES:
        assert uk not in everything, f"{uk} is in ca-qa's templates: a per-country value is hardcoded"
    for uk_env in countries.ENVIRONMENTS:
        assert uk_env.account not in everything
    # And the second country's own values are there.
    for ca in ["America/Toronto", "en-CA", '"CA"']:
        assert ca in everything


def test_the_stacks_are_in_the_entrys_account_and_region(lambda_code):
    app = cdk.App()
    stage = PoisedStage(app, CA_QA, lambda_code=str(lambda_code))
    for stack in (stage.api, stage.web, stage.auth):
        assert stack.region == "ca-central-1"
        assert stack.account == "111111111111"
        assert stack.stack_name.startswith("poised-ca-qa-")


def lambda_env(templates, name):
    fns = templates["api"].find_resources(
        "AWS::Lambda::Function", {"Properties": {"FunctionName": name}}
    )
    assert len(fns) == 1, f"no function {name}"
    return next(iter(fns.values()))["Properties"]


def test_two_arm64_go_functions(lambda_code):
    t = synth(CA_QA, lambda_code)
    for fn in ("staff-api", "patient-api"):
        props = lambda_env(t, f"poised-ca-qa-{fn}")
        assert props["Runtime"] == "provided.al2023"
        assert props["Architectures"] == ["arm64"]
        assert props["Handler"] == "bootstrap"
        assert props["MemorySize"] == 512
        env = props["Environment"]["Variables"]
        assert env["CONFIG_STRICT"] == "true"
        assert env["DATABASE_URL_PARAM"] == "/sj/qa/database-url"
        assert env["APP_URL_PARAM"] == "/sj/qa/app-url"
        assert "TOKEN_SECRET_ARN" in env
        assert env["MESSAGING"] == "capture"
        assert env["DEFAULT_TIME_ZONE"] == "America/Toronto"
        assert env["DEFAULT_LOCALE"] == "en-CA"
        assert env["PHONE_COUNTRY"] == "CA"
        assert env["NATIONAL_ID_TYPE"] == "nhs"
        assert "DEV_LOGIN" not in env  # S12: Cognito on AWS
        # The staff user pool, written by AuthStack (after WebStack: its callback needs the web
        # address), read at cold start.
        assert env["STAFF_POOL_ID_PARAM"] == "/sj/qa/cognito-user-pool-id"
        assert env["STAFF_POOL_CLIENT_ID_PARAM"] == "/sj/qa/cognito-client-id"
        assert env["STAFF_POOL_DOMAIN_PARAM"] == "/sj/qa/cognito-domain"
        assert int(env["FORWARDED_HOPS"]) >= 1
        # Secrets are named, never written into the template.
        for secret in ("DATABASE_URL", "TOKEN_SECRET", "APP_URL"):
            assert secret not in env


def test_logs_are_kept_14_days(lambda_code):
    t = synth(CA_QA, lambda_code)
    t["api"].resource_count_is("AWS::Logs::LogGroup", 3)  # staff-api, patient-api, jobs (S17)
    t["api"].all_resources_properties("AWS::Logs::LogGroup", {"RetentionInDays": 14})


def test_the_token_secret_is_generated_by_cdk(lambda_code):
    t = synth(CA_QA, lambda_code)
    t["api"].has_resource(
        "AWS::SecretsManager::Secret",
        {
            "Properties": {"GenerateSecretString": Match.object_like({"ExcludePunctuation": True})},
            "DeletionPolicy": "Delete",
        },
    )


def policy_statements(t: Template) -> list[dict]:
    out = []
    for p in t.find_resources("AWS::IAM::Policy").values():
        out += p["Properties"]["PolicyDocument"]["Statement"]
    return out


def test_functions_read_only_their_own_parameters(lambda_code):
    t = synth(CA_QA, lambda_code)
    ssm = [s for s in policy_statements(t["api"]) if "ssm:GetParameter" in json.dumps(s["Action"])]
    assert ssm, "the functions can't read their parameters"
    for s in ssm:
        resources = json.dumps(s["Resource"])
        assert "*" not in resources  # no wildcards
        assert "/sj/qa/database-url" in resources and "/sj/qa/app-url" in resources
        for name in ("user-pool-id", "client-id", "domain"):
            assert f"/sj/qa/cognito-{name}" in resources


def test_http_api_routes_the_patient_link_to_patient_api(lambda_code):
    t = synth(CA_QA, lambda_code)["api"]
    t.has_resource_properties("AWS::ApiGatewayV2::Api", {"ProtocolType": "HTTP"})
    routes = {r["Properties"]["RouteKey"] for r in t.find_resources("AWS::ApiGatewayV2::Route").values()}
    assert routes == {"ANY /api/p/{proxy+}", "ANY /api/{proxy+}"}
    # Each route's integration is the right function (payload v2).
    t.all_resources_properties(
        "AWS::ApiGatewayV2::Integration", {"PayloadFormatVersion": "2.0", "IntegrationType": "AWS_PROXY"}
    )
    tpl = t.to_json()
    for route in t.find_resources("AWS::ApiGatewayV2::Route").values():
        integration_id = route["Properties"]["Target"]["Fn::Join"][1][1]["Ref"]
        uri = json.dumps(tpl["Resources"][integration_id]["Properties"]["IntegrationUri"])
        want = "patientapi" if route["Properties"]["RouteKey"].startswith("ANY /api/p/") else "staffapi"
        assert want in uri.lower().replace("-", ""), (route["Properties"]["RouteKey"], uri)


def test_amplify_app_with_rules_in_order(lambda_code):
    t = synth(CA_QA, lambda_code)["web"]
    apps = t.find_resources("AWS::Amplify::App")
    assert len(apps) == 1
    props = next(iter(apps.values()))["Properties"]
    assert "Repository" not in props  # manual deployments: no repository connected
    rules = props["CustomRules"]
    assert len(rules) == 2
    api, spa = rules
    assert api["Source"] == "/api/<*>" and api["Status"] == "200"
    assert "/api/<*>" in json.dumps(api["Target"])  # the HTTP API's address + /api/<*>
    assert spa["Source"] == SPA_REWRITE_SOURCE and spa["Target"] == "/index.html" and spa["Status"] == "200"
    t.has_resource_properties("AWS::Amplify::Branch", {"BranchName": "main", "EnableAutoBuild": False})


def test_the_web_address_is_written_where_the_functions_read_it(lambda_code):
    t = synth(CA_QA, lambda_code)["web"]
    t.has_resource_properties("AWS::SSM::Parameter", {"Name": "/sj/qa/app-url", "Type": "String"})


def test_a_fixed_web_address_is_used_when_the_entry_has_one(lambda_code):
    env = dataclasses.replace(CA_QA, app_url="https://ca.example.test")
    t = synth(env, lambda_code)["web"]
    t.has_resource_properties(
        "AWS::SSM::Parameter", {"Name": "/sj/qa/app-url", "Value": "https://ca.example.test"}
    )


# The SPA rule (Amplify's regex form): a page's address serves index.html; a file, above all a
# missing /assets/ file after a deploy, is a real 404 (pending.md, S04 verification 2).
def amplify_regex(source: str) -> re.Pattern:
    assert source.startswith("</") and source.endswith("/>")
    return re.compile(source[2:-2])


@pytest.mark.parametrize(
    "path",
    [
        "/",
        "/login",
        "/o/7c1e/h/9a2b/questionnaires",
        "/o/7c1e/h/9a2b/episodes/1f2e/validate",
        "/i/AbC_dEf-123",
        "/home/episodes/0d9f",
        "/qa/messages",
        "/s/tok.with.dots",  # a page whose last part has a dot but no file extension
    ],
)
def test_spa_rule_serves_pages(path):
    assert amplify_regex(SPA_REWRITE_SOURCE).search(path), path


@pytest.mark.parametrize(
    "path",
    [
        "/assets/index-abc123.js",
        "/assets/Editor-abc123.css",
        "/assets/gone",  # anything under /assets/ is a file
        "/favicon.ico",
        "/robots.txt",
        "/logo.svg",
        "/fonts/x.woff2",
    ],
)
def test_spa_rule_leaves_files_alone(path):
    assert not amplify_regex(SPA_REWRITE_SOURCE).search(path), path


def test_budget_alarm_for_the_environments_own_resources(lambda_code):
    t = synth(CA_QA, lambda_code)["api"]
    t.has_resource_properties(
        "AWS::Budgets::Budget",
        {
            "Budget": Match.object_like(
                {
                    "BudgetType": "COST",
                    "TimeUnit": "MONTHLY",
                    "BudgetLimit": {"Amount": CA_QA.budget_usd, "Unit": "USD"},
                    "CostFilters": {"TagKeyValue": ["user:poised-env$ca-qa"]},
                }
            ),
        },
    )
    budget = next(iter(t.find_resources("AWS::Budgets::Budget").values()))["Properties"]
    for n in budget["NotificationsWithSubscribers"]:
        assert n["Subscribers"][0]["SubscriptionType"] == "EMAIL"
        # The address is read from SSM at deploy time, never written in the repo.
        assert "Ref" in n["Subscribers"][0]["Address"]
    params = t.to_json()["Parameters"]
    assert any(p.get("Default") == "/sj/alert-email" for p in params.values())


def test_every_resource_is_tagged_with_its_environment(lambda_code):
    """The budget counts the environment's costs by this tag."""
    t = synth(CA_QA, lambda_code)
    tagged = (
        t["api"].find_resources("AWS::Lambda::Function")
        | t["api"].find_resources("AWS::SecretsManager::Secret")
        | t["web"].find_resources("AWS::Amplify::App")
    )
    assert len(tagged) == 5  # 3 functions, the secret, the app
    for r in tagged.values():
        assert {"Key": "poised-env", "Value": "ca-qa"} in r["Properties"]["Tags"], r


def test_no_stack_turns_the_dev_login_on(lambda_code):
    """S12: QA too signs staff in with Cognito, so nothing on AWS works without credentials."""
    for env in (CA_QA, dataclasses.replace(CA_QA, name="ca-training", stage="training", messaging="send")):
        assert "DEV_LOGIN" not in text(synth(env, lambda_code))
    assert "/sj/training/database-url" in text(synth(dataclasses.replace(CA_QA, name="ca-training", stage="training", messaging="send"), lambda_code))


def test_the_real_entries_synthesise(lambda_code):
    app = cdk.App()
    for env in countries.ENVIRONMENTS:
        PoisedStage(app, env, lambda_code=str(lambda_code))
    app.synth()


def test_stacks_use_poiseds_own_cdk_bootstrap(lambda_code):
    """Poised has its own bootstrap (qualifier `poised`, stack poised-cdk-toolkit), so it never
    depends on or changes another project's CDKToolkit in a shared account."""
    for t in synth(CA_QA, lambda_code).values():
        assert t.to_json()["Parameters"]["BootstrapVersion"]["Default"] == "/cdk-bootstrap/poised/version"


# --- The web app's security headers (S12) --------------------------------------------------------


def test_amplify_sends_security_headers_on_every_page(lambda_code):
    app = next(iter(synth(CA_QA, lambda_code)["web"].find_resources("AWS::Amplify::App").values()))
    headers = app["Properties"]["CustomHeaders"]
    assert "pattern: '**'" in headers
    for want in [
        "key: Strict-Transport-Security",
        "value: max-age=31536000; includeSubDomains",
        "key: X-Frame-Options",
        "value: DENY",
        "key: Content-Security-Policy",
        "value: frame-ancestors 'none'",
        "key: X-Content-Type-Options",
        "value: nosniff",
        "key: Referrer-Policy",
        "value: no-referrer",
    ]:
        assert want in headers, want


def test_pages_are_never_kept_for_back_after_sign_out(lambda_code):
    """S12 test report 1, finding 1: on a shared workstation, Back after sign-out showed the
    previous clinician's patient list from the browser's cache. Pages are never stored; the
    hashed files under /assets/ (their names change with their content) are kept for good."""
    headers = next(iter(synth(CA_QA, lambda_code)["web"].find_resources("AWS::Amplify::App").values()))[
        "Properties"
    ]["CustomHeaders"]
    every, assets = headers.split("  - pattern: ")[1:]
    assert every.startswith("'**'")
    assert "- key: Cache-Control\n        value: no-store\n" in every
    assert assets.startswith("'assets/**'")
    assert "- key: Cache-Control\n        value: public, max-age=31536000, immutable\n" in assets


# --- AuthStack: the staff user pool (S12) --------------------------------------------------------


def pool(t: Template) -> dict:
    pools = t.find_resources("AWS::Cognito::UserPool")
    assert len(pools) == 1
    return next(iter(pools.values()))


def test_the_staff_pool(lambda_code):
    p = pool(synth(CA_QA, lambda_code)["auth"])
    props = p["Properties"]
    assert props["UserPoolName"] == "poised-ca-qa-staff"
    # The cheapest plan with managed login (Lite has only the classic hosted UI): free to 10,000 MAUs.
    assert props["UserPoolTier"] == "ESSENTIALS"
    assert props["UsernameAttributes"] == ["email"]  # email is the username
    assert props["AdminCreateUserConfig"]["AllowAdminCreateUserOnly"] is True  # no self sign-up
    assert props["MfaConfiguration"] == "ON"  # required
    assert props["EnabledMfas"] == ["SOFTWARE_TOKEN_MFA"]  # an authenticator app only
    assert props["AccountRecoverySetting"] == {"RecoveryMechanisms": [{"Name": "verified_email", "Priority": 1}]}
    # Cognito's own email (no SES needed, so the SES sandbox doesn't matter): invitations and resets.
    assert props["EmailConfiguration"]["EmailSendingAccount"] == "COGNITO_DEFAULT"
    policy = props["Policies"]["PasswordPolicy"]
    assert policy["TemporaryPasswordValidityDays"] == 7  # as the invite's deadline (S09)
    assert policy["MinimumLength"] >= 12
    # The invitation names the app's address and the temporary password.
    invite = props["AdminCreateUserConfig"]["InviteMessageTemplate"]
    body = json.dumps(invite["EmailMessage"])
    assert "{username}" in body and "{####}" in body
    assert "https://main." in body  # Amplify's address (no domain yet), from WebStack
    # The email is the trusted identity: users can't change it.
    email = [a for a in props["Schema"] if a["Name"] == "email"][0]
    assert email["Mutable"] is False and email["Required"] is True
    # QA's pool goes with its stack, so an account can be emptied and re-created (S06 answers).
    assert p["DeletionPolicy"] == "Delete"
    assert {"Key": "poised-env", "Value": "ca-qa"} in [
        {"Key": k, "Value": v} for k, v in props["UserPoolTags"].items()
    ]


def test_a_real_stages_pool_is_kept(lambda_code):
    t = synth(dataclasses.replace(CA_QA, name="ca-production", stage="production", messaging="send"), lambda_code)["auth"]
    p = pool(t)
    assert p["DeletionPolicy"] == "Retain"
    assert p["Properties"]["DeletionProtection"] == "ACTIVE"


def test_managed_login_on_a_cognito_domain(lambda_code):
    t = synth(CA_QA, lambda_code)["auth"]
    domain = next(iter(t.find_resources("AWS::Cognito::UserPoolDomain").values()))["Properties"]
    assert domain["ManagedLoginVersion"] == 2  # managed login, not the classic hosted UI
    assert re.fullmatch(r"poised-ca-qa-[0-9a-f]{8}", domain["Domain"])
    t.resource_count_is("AWS::Cognito::ManagedLoginBranding", 1)


def client(t: Template) -> dict:
    clients = t.find_resources("AWS::Cognito::UserPoolClient")
    assert len(clients) == 1
    return next(iter(clients.values()))["Properties"]


def test_the_web_app_client(lambda_code):
    c = client(synth(CA_QA, lambda_code)["auth"])
    assert c["GenerateSecret"] is False  # a public client: the code exchange is protected by PKCE
    assert c["AllowedOAuthFlows"] == ["code"]
    assert c["AllowedOAuthFlowsUserPoolClient"] is True
    assert sorted(c["AllowedOAuthScopes"]) == ["email", "openid"]
    assert c["SupportedIdentityProviders"] == ["COGNITO"]
    assert c["PreventUserExistenceErrors"] == "ENABLED"
    assert c["WriteAttributes"] == []  # nobody changes their email through it
    callbacks, logouts = json.dumps(c["CallbackURLs"]), json.dumps(c["LogoutURLs"])
    assert "/auth/callback" in callbacks and "/login" in logouts
    # QA only: the test users' command signs in with a password to set up their authenticator.
    assert "ALLOW_ADMIN_USER_PASSWORD_AUTH" in c["ExplicitAuthFlows"]


def test_only_qa_lets_a_password_sign_in_skip_the_managed_login(lambda_code):
    t = synth(dataclasses.replace(CA_QA, name="ca-training", stage="training", messaging="send"), lambda_code)["auth"]
    assert "ALLOW_ADMIN_USER_PASSWORD_AUTH" not in client(t)["ExplicitAuthFlows"]


def test_a_fixed_web_address_is_the_callback(lambda_code):
    t = synth(dataclasses.replace(CA_QA, app_url="https://ca.example.test"), lambda_code)["auth"]
    c = client(t)
    assert c["CallbackURLs"] == ["https://ca.example.test/auth/callback"]
    assert c["LogoutURLs"] == ["https://ca.example.test/login"]


def test_the_pool_is_written_where_the_functions_read_it(lambda_code):
    t = synth(CA_QA, lambda_code)["auth"]
    names = {p["Properties"]["Name"] for p in t.find_resources("AWS::SSM::Parameter").values()}
    assert names == {"/sj/qa/cognito-user-pool-id", "/sj/qa/cognito-client-id", "/sj/qa/cognito-domain"}


def test_staff_api_can_invite_into_this_pool_only(lambda_code):
    t = synth(CA_QA, lambda_code)["auth"]
    statements = policy_statements(t)
    assert len(statements) == 1
    [s] = statements
    assert s["Action"] == "cognito-idp:AdminCreateUser"
    assert "StaffPool" in json.dumps(s["Resource"])  # the pool's own ARN, no wildcard
    [policy] = t.find_resources("AWS::IAM::Policy").values()
    assert "staffapi" in json.dumps(policy["Properties"]["Roles"]).lower().replace("-", "")
