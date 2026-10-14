"""The environment's monthly budget alarm (plan F4 Step 0, moved into CDK on 14 Oct 2026).

It counts only the environment's own costs, by the `poised-env` tag every resource carries (the
account may host other projects). The alert address is read from SSM (/sj/alert-email, stored by
`make aws-setup`) when the stack deploys, so no address is written in the repo."""

from __future__ import annotations

from aws_cdk import aws_budgets as budgets
from aws_cdk import aws_ssm as ssm
from constructs import Construct

from countries import Environment

ALERT_EMAIL_PARAM = "/sj/alert-email"
COST_TAG = "poised-env"


class Budget(Construct):
    def __init__(self, scope: Construct, id: str, *, env: Environment) -> None:
        super().__init__(scope, id)
        email = ssm.StringParameter.value_for_string_parameter(self, ALERT_EMAIL_PARAM)

        def alert(kind: str, percent: int) -> budgets.CfnBudget.NotificationWithSubscribersProperty:
            return budgets.CfnBudget.NotificationWithSubscribersProperty(
                notification=budgets.CfnBudget.NotificationProperty(
                    notification_type=kind,
                    comparison_operator="GREATER_THAN",
                    threshold=percent,
                    threshold_type="PERCENTAGE",
                ),
                subscribers=[budgets.CfnBudget.SubscriberProperty(subscription_type="EMAIL", address=email)],
            )

        budgets.CfnBudget(
            self,
            "Monthly",
            budget=budgets.CfnBudget.BudgetDataProperty(
                budget_name=f"poised-{env.name}",
                budget_type="COST",
                time_unit="MONTHLY",
                budget_limit=budgets.CfnBudget.SpendProperty(amount=env.budget_usd, unit="USD"),
                cost_filters={"TagKeyValue": [f"user:{COST_TAG}${env.name}"]},
            ),
            notifications_with_subscribers=[alert("ACTUAL", 80), alert("FORECASTED", 100)],
        )
