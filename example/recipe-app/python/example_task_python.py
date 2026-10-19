import base64
import binascii
import json
from typing import Any, Dict


def NewExampleTaskPython(**kwargs) -> "ExampleTaskPython":
    return ExampleTaskPython()


class ExampleTaskPython:
    def __init__(self, **kwargs):
        self.message = kwargs.get('message', 'a little extra love')

    def name(self) -> str:
        return "exampleTaskPython"

    def apply(self, context: Dict[str, Any], request: Dict[str, Any]) -> None:
        """Add the example ingredient to a ServiceRequest's base64 ``Body``."""
        if not request or 'Body' not in request or not request['Body']:
            raise ValueError("Invalid request or empty Body")

        try:
            body = base64.b64decode(request['Body'], validate=True)
        except (binascii.Error, ValueError):
            raise ValueError("Body must be valid base64") from None

        # Apply the changes using the execute method
        result = self.execute(body)

        # Preserve the ServiceRequest wire representation.
        request['Body'] = base64.b64encode(result).decode('ascii')

    def execute(self, input_data: bytes) -> bytes:
        # Deserialize recipe
        try:
            recipe = json.loads(input_data.decode('utf-8'))
        except UnicodeDecodeError:
            raise ValueError("Body must contain UTF-8 JSON") from None
        except json.JSONDecodeError:
            raise ValueError("Body must contain valid JSON") from None

        if not isinstance(recipe, dict):
            raise ValueError("Recipe must be a JSON object")

        # Add "a little extra love" as an ingredient
        if 'ingredients' not in recipe:
            recipe['ingredients'] = []
        elif not isinstance(recipe['ingredients'], list):
            raise ValueError("Recipe 'ingredients' must be a list")
        recipe['ingredients'].append("a little extra love")

        # Serialize the updated recipe
        try:
            updated_recipe_json = json.dumps(recipe).encode('utf-8')
        except Exception as e:
            raise ValueError(f"Failed to marshal updated recipe: {str(e)}")

        return updated_recipe_json

# Example usage:
if __name__ == "__main__":
    task = ExampleTaskPython()
    sample_recipe = {
        'name': 'Chocolate Cake',
        'ingredients': ['flour', 'sugar', 'cocoa powder'],
        'steps': ['Mix dry ingredients', 'Add wet ingredients', 'Bake'],
        'comments': [{'text': 'Delicious!'}]
    }
    sample_request = {
        'Body': base64.b64encode(
            json.dumps(sample_recipe).encode('utf-8')
        ).decode('ascii')
    }

    task.apply({}, sample_request)
    print(json.loads(base64.b64decode(sample_request['Body'])))


def example_task_python(request: Dict[str, Any]) -> Dict[str, Any]:
    """Module-level entry point loaded by integrations/python/handler_loader.py."""
    ExampleTaskPython().apply({}, request)
    return request
