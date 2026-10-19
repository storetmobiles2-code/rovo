#!/bin/sh
# SPDX-License-Identifier: Apache-2.0
# Creates the rovo buckets in the local S3-compatible store (idempotent).
#   media : public-read (anonymous GET of objects; served same-origin at /media/*)
#   kyc   : private (presigned URLs only), versioned like prod
set -eu

endpoint="${MINIO_ENDPOINT:-http://minio:9000}"
media="${ROVO_S3_BUCKET_MEDIA:-media}"
kyc="${ROVO_S3_BUCKET_KYC:-kyc}"

i=0
until mc alias set rovo "$endpoint" "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null 2>&1; do
  i=$((i + 1))
  if [ "$i" -ge 30 ]; then
    echo "minio-init: cannot reach $endpoint" >&2
    exit 1
  fi
  sleep 1
done

mc mb --ignore-existing --region ap-south-1 "rovo/$media"
mc mb --ignore-existing --region ap-south-1 "rovo/$kyc"

# Public read for media objects only (no anonymous write).
mc anonymous set download "rovo/$media"
# KYC stays private; make that explicit in case it was ever changed.
mc anonymous set none "rovo/$kyc"
mc version enable "rovo/$kyc" || true

echo "minio-init: buckets ready: $media (public-read), $kyc (private)"
