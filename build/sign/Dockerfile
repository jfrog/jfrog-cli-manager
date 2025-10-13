# Docker container for signing Windows binaries
FROM ubuntu:20.04

# Avoid interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install dependencies
RUN apt-get update && apt-get install -y \
    osslsigncode \
    curl \
    wget \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Create working directory
WORKDIR /workspace

# Create a non-root user for security
RUN groupadd -r signer && useradd -r -g signer -d /workspace -s /bin/bash signer
RUN chown -R signer:signer /workspace

# Copy signing script
COPY sign-windows.sh /usr/local/bin/sign-windows.sh
RUN chmod +x /usr/local/bin/sign-windows.sh

# Switch to non-root user
USER signer

# Set entrypoint
ENTRYPOINT ["/usr/local/bin/sign-windows.sh"]


