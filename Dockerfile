# =============================================================================
# Build stage: compile Go binary
# =============================================================================
FROM golang:1.26-bookworm AS builder

WORKDIR /src

# Download dependencies first (Docker cache optimization)
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build with best practices:
# - CGO_ENABLED=0: static binary
# - trimpath: reproducible builds
# - ldflags -s -w: strip debug info, reduce size
ARG VERSION=dev
RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o /picoloom ./cmd/picoloom

# =============================================================================
# Runtime stage: minimal image with headless Chromium
# =============================================================================
FROM chromedp/headless-shell:stable

# Install CA certificates, fontconfig, emoji, and fonts
# - fonts-liberation: Metric-compatible with Arial, Times New Roman, Courier New
# - fonts-texgyre: High-quality alternatives (TeX Gyre Schola for Georgia)
# - Source Sans 3: Light, modern sans-serif for default/technical/creative styles
# - JetBrains Mono: Excellent monospace for code blocks
# - fonts-noto-color-emoji: Color emoji support
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    fontconfig \
    fonts-liberation \
    fonts-texgyre \
    fonts-noto-color-emoji \
    curl \
    unzip \
    && rm -rf /var/lib/apt/lists/* \
    # Download and install Source Sans 3 font
    && curl -sL "https://github.com/adobe-fonts/source-sans/releases/download/3.052R/OTF-source-sans-3.052R.zip" -o /tmp/sourcesans.zip \
    && unzip -q /tmp/sourcesans.zip -d /tmp/sourcesans \
    && mkdir -p /usr/share/fonts/opentype/source-sans-3 \
    && cp /tmp/sourcesans/OTF/*.otf /usr/share/fonts/opentype/source-sans-3/ \
    && rm -rf /tmp/sourcesans /tmp/sourcesans.zip \
    # Download and install JetBrains Mono font
    && curl -sL https://github.com/JetBrains/JetBrainsMono/releases/download/v2.304/JetBrainsMono-2.304.zip -o /tmp/jbmono.zip \
    && unzip -q /tmp/jbmono.zip -d /tmp/jbmono \
    && mkdir -p /usr/share/fonts/truetype/jetbrains-mono \
    && cp /tmp/jbmono/fonts/ttf/*.ttf /usr/share/fonts/truetype/jetbrains-mono/ \
    && rm -rf /tmp/jbmono /tmp/jbmono.zip \
    # Rebuild font cache
    && fc-cache -fv \
    # Remove curl/unzip (no longer needed)
    && apt-get purge -y curl unzip \
    && apt-get autoremove -y

# Copy binary and keep a legacy alias for existing scripts
COPY --from=builder /picoloom /usr/bin/picoloom
RUN ln -sf /usr/bin/picoloom /usr/bin/md2pdf

# Create non-root user with home directory (rod needs it for cache)
RUN useradd -r -u 1000 -m -s /bin/false appuser

# Rod configuration:
# - ROD_BROWSER_BIN: use pre-installed headless Chrome
# - ROD_NO_SANDBOX: Docker lacks kernel capabilities for Chrome sandboxing
ENV ROD_BROWSER_BIN=/headless-shell/headless-shell \
    ROD_NO_SANDBOX=1

USER appuser

# Working directory for files
WORKDIR /data

ENTRYPOINT ["/usr/bin/picoloom"]
