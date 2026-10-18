FROM python:3.12-slim
RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY integrations/python/requirements.txt ./requirements.txt
RUN pip install --no-cache-dir -r requirements.txt
COPY api/schema ./api/schema
COPY example/recipe-app/python ./example/recipe-app/python
COPY integrations/python ./integrations/python
ENV PROJECT_ROOT=/app \
    YAML_FILE_PATH=/app/example/recipe-app/python/handlers.yaml \
    PORT=5000
EXPOSE 5000
HEALTHCHECK --interval=5s --timeout=3s --retries=10 CMD curl -fsS http://localhost:5000/health || exit 1
CMD ["python", "integrations/python/start.py"]
