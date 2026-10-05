# n8n + FFmpeg + the Python/OpenCV finishing engine, in one image.
# NOTE: written for you, but NOT built in my sandbox (no Docker daemon there). The same install steps
# (n8n 2.35.7, ffmpeg, python3-opencv/numpy/scipy/pillow) were exercised natively and all 3 workflows ran.
FROM node:22-bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
      ffmpeg python3 python3-numpy python3-scipy python3-opencv python3-pil ca-certificates tini \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g n8n@2.35.7 --no-audit --no-fund

# n8n 2.x switches off shell access + restricts file access by default; this pipeline needs both (inside this container only).
ENV NODES_EXCLUDE="[]" \
    N8N_RESTRICT_FILE_ACCESS_TO="/data" \
    N8N_BLOCK_FILE_ACCESS_TO_N8N_FILES="true" \
    N8N_RUNNERS_ENABLED="true" \
    N8N_DIAGNOSTICS_ENABLED="false" \
    N8N_VERSION_NOTIFICATIONS_ENABLED="false" \
    GENERIC_TIMEZONE="Asia/Kolkata" \
    N8N_USER_FOLDER="/home/node"

RUN mkdir -p /data && chown -R node:node /data /home/node
USER node
WORKDIR /data
EXPOSE 5678
ENTRYPOINT ["tini", "--"]
CMD ["n8n", "start"]
