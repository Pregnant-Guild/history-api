FROM postgis/postgis:18-3.6

# Install build dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
    build-essential \
    postgresql-server-dev-18 \
    git \
    ca-certificates && \
    rm -rf /var/lib/apt/lists/*

# Build and install pgvector from source
RUN cd /tmp && \
    git clone --branch v0.8.2 https://github.com/pgvector/pgvector.git && \
    cd pgvector && \
    make && \
    make install && \
    rm -rf /tmp/pgvector

# Cleanup
RUN apt-get purge -y --auto-remove build-essential postgresql-server-dev-18 git
