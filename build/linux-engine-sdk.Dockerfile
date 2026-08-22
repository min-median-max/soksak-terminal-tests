FROM ubuntu:24.04

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    build-essential ca-certificates clang-20 curl glslang-tools libcanberra-dev libcairo2-dev \
    libdbus-1-dev libfontconfig-dev libgl1-mesa-dev libharfbuzz-dev liblcms2-dev libpng-dev \
    libpython3-dev libsimde-dev libsystemd-dev libvulkan-dev libwayland-dev libx11-xcb-dev libxcb-xkb-dev \
    libxcursor-dev libxi-dev libxinerama-dev libxkbcommon-dev libxkbcommon-x11-dev \
    libxrandr-dev librsvg2-bin libssl-dev libxxhash-dev patchelf pkg-config python3 python3-dev ragel \
    python3-full spirv-cross uuid-dev wayland-protocols \
    && rm -rf /var/lib/apt/lists/*

RUN python3 -c "import encodings, json"
RUN clang++-20 --version && printf 'int main(){}' | clang++-20 -x c++ -std=c++26 -fsyntax-only -

ENV CC=clang-20
ENV CXX=clang++-20

WORKDIR /source
