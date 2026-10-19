import base64
import copy
import json
import os
import tempfile
import unittest

from handler_loader import DynamicHandlerImporter
from request_handler import create_app
from utils import find_project_root


class TestExampleTaskPythonContract(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.project_root = find_project_root()
        handler_path = os.path.join(
            cls.project_root, "example", "recipe-app", "python"
        )
        with tempfile.NamedTemporaryFile(
            mode="w", suffix=".yaml", delete=False
        ) as config_file:
            config_file.write(
                "- unique_name: exampleTaskPython\n"
                "  handler_name: example_task_python\n"
                f"  handler_path: {handler_path}\n"
            )
            cls.config_path = config_file.name

        handlers = DynamicHandlerImporter(cls.config_path).load_handlers()
        if "exampleTaskPython" not in handlers:
            raise AssertionError("real exampleTaskPython entry point did not load")
        cls.client = create_app(handlers).test_client()

    @classmethod
    def tearDownClass(cls):
        os.unlink(cls.config_path)

    def service_request(self, body_marker=...):
        request = {
            "ApiName": "recipe-api",
            "ServiceName": "recipe-service",
            "Method": "POST",
            "URL": "http://example.com/recipes/42",
            "InternalPath": "/recipes/42",
            "Header": {"X-Correlation-ID": ["synthetic-42"]},
            "Form": {"mode": ["test"]},
            "RequestMeta": {
                "Params": {"recipe_id": "42"},
                "Proto": "HTTP/1.1",
                "ProtoMajor": 1,
                "ProtoMinor": 1,
                "ContentLength": 0,
                "Host": "example.com",
                "RemoteAddr": "127.0.0.1",
                "RequestURI": "/recipes/42",
            },
        }
        if body_marker is not ...:
            request["Body"] = body_marker
        return request

    def post(self, request):
        return self.client.post(
            "/apply/exampleTaskPython",
            json={"Handler": "exampleTaskPython", "Request": request},
        )

    @staticmethod
    def encoded_json(value):
        return base64.b64encode(json.dumps(value).encode("utf-8")).decode("ascii")

    def assert_handler_error(self, request, message):
        response = self.post(request)
        self.assertEqual(response.status_code, 500)
        self.assertEqual(
            response.json, {"error": f"Handler execution error: {message}"}
        )

    def test_schema_body_round_trips_and_preserves_metadata(self):
        recipe = {"name": "Synthetic Soup", "ingredients": ["water"]}
        request = self.service_request(self.encoded_json(recipe))
        expected_metadata = copy.deepcopy(request)
        del expected_metadata["Body"]

        response = self.post(request)

        self.assertEqual(response.status_code, 200)
        result = response.json
        encoded_body = result.pop("Body")
        self.assertEqual(result, expected_metadata)
        self.assertIsInstance(encoded_body, str)
        decoded_recipe = json.loads(base64.b64decode(encoded_body, validate=True))
        self.assertEqual(
            decoded_recipe,
            {
                "name": "Synthetic Soup",
                "ingredients": ["water", "a little extra love"],
            },
        )

    def test_missing_or_empty_body_is_rejected(self):
        for body_marker in (..., ""):
            with self.subTest(body_marker=body_marker):
                self.assert_handler_error(
                    self.service_request(body_marker),
                    "Invalid request or empty Body",
                )

    def test_malformed_base64_is_rejected(self):
        self.assert_handler_error(
            self.service_request("not base64!!"), "Body must be valid base64"
        )

    def test_non_utf8_body_is_rejected(self):
        body = base64.b64encode(b"\xff").decode("ascii")
        self.assert_handler_error(
            self.service_request(body), "Body must contain UTF-8 JSON"
        )

    def test_invalid_json_body_is_rejected(self):
        body = base64.b64encode(b"not json").decode("ascii")
        self.assert_handler_error(
            self.service_request(body), "Body must contain valid JSON"
        )

    def test_unsuitable_recipe_shapes_are_rejected(self):
        cases = (
            (["not", "an", "object"], "Recipe must be a JSON object"),
            (
                {"name": "Synthetic Soup", "ingredients": "water"},
                "Recipe 'ingredients' must be a list",
            ),
        )
        for recipe, message in cases:
            with self.subTest(recipe=recipe):
                self.assert_handler_error(
                    self.service_request(self.encoded_json(recipe)), message
                )


if __name__ == "__main__":
    unittest.main()
