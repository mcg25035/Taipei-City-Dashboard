FROM python:3.11-slim

WORKDIR /app

# 安裝系統依賴
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    libpq-dev \
    curl \
    && rm -rf /var/lib/apt/lists/*

# 複製並安裝 Python 依賴
COPY requirements.txt .
RUN pip install --no-cache-dir --retries 10 --timeout 300 \
    torch --index-url https://download.pytorch.org/whl/cpu \
 && pip install --no-cache-dir --retries 10 --timeout 300 -r requirements.txt

# 複製腳本
COPY upgrade_vector_db.py .

# 預設指令
CMD ["python", "upgrade_vector_db.py"]
