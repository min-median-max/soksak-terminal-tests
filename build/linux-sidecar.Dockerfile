FROM rust:1.96-bookworm AS rust-toolchain

FROM ubuntu:24.04

COPY --from=rust-toolchain /usr/local/cargo /usr/local/cargo
COPY --from=rust-toolchain /usr/local/rustup /usr/local/rustup

ENV PATH=/usr/local/cargo/bin:$PATH
ENV CARGO_HOME=/usr/local/cargo
ENV RUSTUP_HOME=/usr/local/rustup

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates clang g++ git libxxhash-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*

ENV CARGO_NET_GIT_FETCH_WITH_CLI=true
RUN rustc --version | grep -F 'rustc 1.96.'
WORKDIR /workspace
