FROM python:3.12-slim

# Install system dependencies (graphviz)
RUN apt-get update && apt-get install -y --no-install-recommends \
    graphviz \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy requirements file first to take advantage of Docker cache
COPY requirements.txt ./

# Install python dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy the rest of the application files
COPY . .

# Expose default port
EXPOSE 60003

# Run the application
CMD ["python", "main.py"]
