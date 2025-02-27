# syntax=docker/dockerfile:1
# Create a stage for building the application.
ARG GO_VERSION=1.24
FROM --platform=$BUILDPLATFORM golang:${GO_VERSION} AS build
WORKDIR /src

# Download dependencies as a separate step to take advantage of Docker's caching.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage bind mounts to go.sum and go.mod to avoid having to copy them into
# the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x

# Build the application.
# Leverage a cache mount to /go/pkg/mod/ to speed up subsequent builds.
# Leverage a bind mount to the current directory to avoid having to copy the
# source code into the container.
RUN --mount=type=cache,target=/go/pkg/mod/ \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 go build -o /bin/dbot ./cmd/dbot/main.go

################################################################################
# Create a new stage for running the application that contains the minimal
FROM alpine:latest AS final

# Install any runtime dependencies that are needed to run your application.
# Leverage a cache mount to /var/cache/apk/ to speed up subsequent builds.
RUN --mount=type=cache,target=/var/cache/apk \
    apk --update add \
    ca-certificates \
    tzdata \
    ffmpeg \ 
    python3 \
    && \
    update-ca-certificates

# isntall ytdlp from GH
RUN wget https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp && \
    chmod +x ./yt-dlp && \
    mv ./yt-dlp /usr/bin

# Create a non-privileged user that the app will run under.
# See https://docs.docker.com/go/dockerfile-user-best-practices/
ARG UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    appuser


WORKDIR /dbot
# Copy the executable from the "build" stage.
COPY --from=build /bin/dbot .
COPY .prod.env ./.env
COPY ./cookies.txt .
RUN chmod +x ./dbot

# USER appuser

# Expose the port that the application listens on.
EXPOSE 58008

# What the container should run when it is started.
# CMD [ "sleep","90900900" ]
CMD [ "/dbot/dbot" ]

