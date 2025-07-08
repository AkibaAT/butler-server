# Install MinIO client (mc)
RUN if [ "$TARGETARCH" = "arm64" ]; then \
        wget -O /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-arm64/mc; \
    else \
        wget -O /usr/local/bin/mc https://dl.min.io/client/mc/release/linux-amd64/mc; \
    fi && \
    chmod +x /usr/local/bin/mc

# Install additional useful tools
RUN apt-get update && apt-get install -y \
    curl \
    jq \
    tree \
    htop \
    postgresql-client \
    && rm -rf /var/lib/apt/lists/*
