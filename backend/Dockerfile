# Build image backend SIMPEG.
#
# Image memuat tiga binary dan dipakai ulang oleh tiga layanan compose:
#   server    -> HTTP API + worker latar (entrypoint bawaan)
#   jobsched  -> penjadwal pekerjaan ke RabbitMQ (entrypoint dioverride compose)
#   migrate   -> penerapan migrasi Goose, sekali jalan (entrypoint dioverride compose)
#
# Dibangun dari direktori backend/:
#   docker build -t simpeg-backend .
#
# Binary bersifat statis (CGO dimatikan) sehingga image akhir memakai
# distroless tanpa shell.
ARG GO_VERSION=1.27.4

FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
ENV GOTOOLCHAIN=local CGO_ENABLED=0
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
    && go build -trimpath -ldflags="-s -w" -o /out/jobsched ./cmd/jobsched \
    && go build -trimpath -ldflags="-s -w" -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/server /bin/server
COPY --from=build /out/jobsched /bin/jobsched
COPY --from=build /out/migrate /bin/migrate
USER nonroot:nonroot
ENTRYPOINT ["/bin/server"]
