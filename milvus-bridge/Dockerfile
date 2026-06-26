FROM python:3.11-slim

WORKDIR /app

RUN pip install --no-cache-dir pymilvus flask

COPY app.py .

EXPOSE 8088

CMD ["python", "app.py"]
