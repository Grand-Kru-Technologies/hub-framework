import os
from handler_loader import DynamicHandlerImporter
from request_handler import create_app


def start(yaml_file_path):
    # Load handlers using DynamicHandlerImporter
    importer = DynamicHandlerImporter(yaml_file_path)
    handlers = importer.load_handlers()

    # Create the Flask app with the loaded handlers
    app = create_app(handlers)

    return app


def server_options(environ=None):
    """Return explicit server settings for local or container startup."""
    environ = os.environ if environ is None else environ
    return {
        'host': environ.get('HOST', '127.0.0.1'),
        'port': int(environ.get('PORT', '5000')),
        'debug': environ.get('FLASK_DEBUG') == '1',
    }


if __name__ == '__main__':
    yaml_file_path = os.getenv('YAML_FILE_PATH')
    app = start(yaml_file_path)
    app.run(**server_options())
