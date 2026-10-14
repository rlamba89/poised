"""AuthStack: the staff user pool (plan C1, session S12). Staff sign in through Cognito's managed
login (password, then an authenticator app code: MFA is required); the web app sends them there
with the OAuth code flow and PKCE, and the Go API exchanges the code and sets its own session
cookie. Accounts are made only by invites (AdminCreateUser, from staff-api): no self sign-up.

It comes after WebStack, because the app client's callback and sign-out addresses are the web
address, which until there's a domain is Amplify's own. So the functions can't be given the pool's
IDs directly; this stack writes them to SSM, where the functions read them at cold start, and
grants staff-api its invite permission itself.
"""

from __future__ import annotations

import hashlib

import aws_cdk as cdk
from aws_cdk import aws_cognito as cognito
from aws_cdk import aws_iam as iam
from aws_cdk import aws_ssm as ssm
from constructs import Construct

from api_stack import COGNITO_PARAMS
from countries import Environment

# How long an invitation's temporary password lasts: the invite's deadline for a first sign-in (S09).
INVITE_DAYS = 7


class AuthStack(cdk.Stack):
    def __init__(
        self, scope: Construct, id: str, *, env: Environment, app_url: str, staff_api_role: iam.IRole, **kwargs
    ) -> None:
        super().__init__(scope, id, **kwargs)
        qa = env.stage == "qa"

        self.pool = cognito.UserPool(
            self,
            "StaffPool",
            user_pool_name=f"poised-{env.name}-staff",
            # The cheapest plan with managed login (Lite has only the classic hosted UI); free up
            # to 10,000 monthly active users (S12, AWS pricing checked 14 Oct 2026).
            feature_plan=cognito.FeaturePlan.ESSENTIALS,
            self_sign_up_enabled=False,
            sign_in_aliases=cognito.SignInAliases(email=True),  # the email is the username
            sign_in_case_sensitive=False,
            # The email is who someone is in Poised (the API finds them by it at the first
            # sign-in), so it can't be changed by them; invites mark it verified.
            standard_attributes=cognito.StandardAttributes(
                email=cognito.StandardAttribute(required=True, mutable=False)
            ),
            mfa=cognito.Mfa.REQUIRED,
            mfa_second_factor=cognito.MfaSecondFactor(otp=True, sms=False, email=False),
            account_recovery=cognito.AccountRecovery.EMAIL_ONLY,
            password_policy=cognito.PasswordPolicy(
                min_length=12,
                require_lowercase=True,
                require_uppercase=True,
                require_digits=True,
                require_symbols=True,
                temp_password_validity=cdk.Duration.days(INVITE_DAYS),
            ),
            # Cognito's own email (no-reply@verificationemail.com, about 50 a day): it needs no SES,
            # so SES's sandbox doesn't stop invitations. A trust's volume needs SES later (pending.md).
            email=cognito.UserPoolEmail.with_cognito(),
            user_invitation=cognito.UserInvitationConfig(
                email_subject="You're invited to Poised",
                email_body=(
                    "You've been invited to Poised, for your hospital's pre-operative assessments.<br><br>"
                    f"Sign in at {app_url} with your email address, {{username}}, and this temporary "
                    "password: <b>{####}</b><br><br>"
                    "You'll choose your own password, then set up an authenticator app (such as Microsoft "
                    f"or Google Authenticator) for the codes you'll sign in with. The temporary password "
                    f"works for {INVITE_DAYS} days; after that, ask your trust's admin to invite you again."
                ),
            ),
            user_verification=cognito.UserVerificationConfig(
                email_subject="Your Poised code",
                email_body="Your Poised code is {####}. If you didn't ask to reset your password, ignore this email.",
                email_style=cognito.VerificationEmailStyle.CODE,
            ),
            # QA's pool goes with its stack, so an account can be emptied and re-created; a real
            # stage's pool (its staff's accounts) is kept and can't be deleted by mistake.
            removal_policy=cdk.RemovalPolicy.DESTROY if qa else cdk.RemovalPolicy.RETAIN,
            deletion_protection=not qa,
        )

        # The managed login's address: https://<prefix>.auth.<region>.amazoncognito.com. Prefixes
        # are unique in a region across every AWS account, so the account (hashed) is in it.
        prefix = f"poised-{env.name}-{hashlib.sha256(env.account.encode()).hexdigest()[:8]}"
        self.domain = self.pool.add_domain(
            "Domain",
            cognito_domain=cognito.CognitoDomainOptions(domain_prefix=prefix),
            managed_login_version=cognito.ManagedLoginVersion.NEWER_MANAGED_LOGIN,
        )

        self.client = self.pool.add_client(
            "WebClient",
            user_pool_client_name=f"poised-{env.name}-web",
            generate_secret=False,  # a browser app: the code exchange is protected by PKCE instead
            # QA only: the test users' command (cmd/qa-users) signs in with a password, with AWS
            # credentials, to set up their authenticators. Never on a real stage.
            auth_flows=cognito.AuthFlow(user_srp=True, admin_user_password=qa),
            o_auth=cognito.OAuthSettings(
                flows=cognito.OAuthFlows(authorization_code_grant=True),
                scopes=[cognito.OAuthScope.OPENID, cognito.OAuthScope.EMAIL],
                callback_urls=[f"{app_url}/auth/callback"],
                logout_urls=[f"{app_url}/login"],
            ),
            supported_identity_providers=[cognito.UserPoolClientIdentityProvider.COGNITO],
            prevent_user_existence_errors=True,
            read_attributes=cognito.ClientAttributes().with_standard_attributes(email=True, email_verified=True),
            write_attributes=cognito.ClientAttributes(),
            id_token_validity=cdk.Duration.minutes(60),
            access_token_validity=cdk.Duration.minutes(60),
            refresh_token_validity=cdk.Duration.hours(1),  # not used: the API keeps its own session
        )

        # Managed login needs a style for the app client; Cognito's own look for now.
        cognito.CfnManagedLoginBranding(
            self,
            "Branding",
            user_pool_id=self.pool.user_pool_id,
            client_id=self.client.user_pool_client_id,
            use_cognito_provided_values=True,
        )

        values = {
            "STAFF_POOL_ID": self.pool.user_pool_id,
            "STAFF_POOL_CLIENT_ID": self.client.user_pool_client_id,
            "STAFF_POOL_DOMAIN": self.domain.base_url(),
        }
        for name, suffix in COGNITO_PARAMS.items():
            ssm.StringParameter(
                self,
                f"Param-{suffix}",
                parameter_name=f"{env.param_prefix}/cognito-{suffix}",
                string_value=values[name],
                description=f"Poised {env.name}: the staff user pool ({name})",
            )

        # Invites (staff-api): AdminCreateUser on this pool only.
        iam.Policy(
            self,
            "StaffApiInvites",
            roles=[staff_api_role],
            statements=[iam.PolicyStatement(actions=["cognito-idp:AdminCreateUser"], resources=[self.pool.user_pool_arn])],
        )

        cdk.CfnOutput(self, "UserPoolId", value=self.pool.user_pool_id)
        cdk.CfnOutput(self, "ManagedLoginUrl", value=self.domain.base_url())
