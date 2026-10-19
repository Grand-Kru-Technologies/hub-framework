import ast
import pathlib
import unittest


REPOSITORY_ROOT = pathlib.Path(__file__).resolve().parents[2]
DEPLOY_WORKFLOW = REPOSITORY_ROOT / ".github" / "workflows" / "deploy.yml"


class DeployWorkflowTest(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        workflow = DEPLOY_WORKFLOW.read_text(encoding="utf-8")
        railway_start = workflow.index("  railway:\n")
        railway_header_end = workflow.index("    runs-on:", railway_start)
        railway_header = workflow[railway_start:railway_header_end]
        condition_start = railway_header.index("    if: >-\n") + len("    if: >-\n")
        condition_lines = railway_header[condition_start:].splitlines()
        cls.condition = " ".join(line.strip() for line in condition_lines)

    def evaluate_condition(
        self,
        *,
        event_name,
        conclusion=None,
        triggering_event=None,
        head_branch=None,
        head_repository=None,
        repository="Grand-Kru-Technologies/hub-framework",
    ):
        values = {
            "github.event.workflow_run.head_repository.full_name": head_repository,
            "github.event.workflow_run.conclusion": conclusion,
            "github.event.workflow_run.head_branch": head_branch,
            "github.event.workflow_run.event": triggering_event,
            "github.event_name": event_name,
            "github.repository": repository,
        }
        expression = self.condition
        for variable in sorted(values, key=len, reverse=True):
            expression = expression.replace(variable, repr(values[variable]))
        expression = expression.replace("&&", " and ").replace("||", " or ")
        tree = ast.parse(expression, mode="eval")
        allowed_nodes = (
            ast.Expression,
            ast.BoolOp,
            ast.And,
            ast.Or,
            ast.Compare,
            ast.Eq,
            ast.Constant,
            ast.Load,
        )
        self.assertTrue(all(isinstance(node, allowed_nodes) for node in ast.walk(tree)))
        return bool(eval(compile(tree, "<workflow-if>", "eval"), {"__builtins__": {}}, {}))

    def test_automatic_deploy_event_matrix(self):
        cases = (
            ("trusted main push", True, "success", "push", "main", "Grand-Kru-Technologies/hub-framework"),
            ("fork main push", False, "success", "push", "main", "someone/hub-framework"),
            ("pull request main", False, "success", "pull_request", "main", "Grand-Kru-Technologies/hub-framework"),
            ("failed CI", False, "failure", "push", "main", "Grand-Kru-Technologies/hub-framework"),
            ("other branch", False, "success", "push", "feature", "Grand-Kru-Technologies/hub-framework"),
        )
        for name, expected, conclusion, event, branch, repository in cases:
            with self.subTest(name=name):
                self.assertEqual(
                    expected,
                    self.evaluate_condition(
                        event_name="workflow_run",
                        conclusion=conclusion,
                        triggering_event=event,
                        head_branch=branch,
                        head_repository=repository,
                    ),
                )

    def test_manual_dispatch_remains_explicit(self):
        self.assertTrue(self.evaluate_condition(event_name="workflow_dispatch"))


if __name__ == "__main__":
    unittest.main()
