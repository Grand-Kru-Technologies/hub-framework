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

if __name__ == '__main__':
    yaml_file_path = os.getenv('YAML_FILE_PATH')
    app = start(yaml_file_path)
    app.run(host=os.getenv('HOST', '0.0.0.0'), port=int(os.getenv('PORT', '5000')), debug=os.getenv('FLASK_DEBUG') == '1')