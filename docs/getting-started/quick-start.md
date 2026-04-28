# Quick Start

This guide is for running MyStravaStats locally without working on the code.

## 1. Install Docker

Docker must be installed and running before you use the build scripts or compose files.

- macOS: [Install Docker Desktop on Mac](https://docs.docker.com/desktop/setup/install/mac-install/)
- Ubuntu: [Install Docker Engine on Ubuntu](https://docs.docker.com/engine/install/ubuntu/)
- Windows: [Install Docker Desktop on Windows](https://docs.docker.com/desktop/setup/install/windows-install/)

## 2. Build A Local Binary

From the repository root:

```sh
./build-go-macos.zsh
```

Linux:

```sh
./build-go-ubuntu.sh
```

Windows:

```powershell
.\build-go-windows.ps1
```

The Go build scripts create `mystravastats` or `mystravastats.exe`.

## 3. Run The Application

macOS/Linux:

```sh
./mystravastats
```

Windows:

```powershell
mystravastats.exe
```

Then open [http://localhost:8080/](http://localhost:8080/).

## 4. Configure Strava

For live Strava synchronization, create a `.strava` file as described in [Strava OAuth Setup](../data-sources/strava-oauth.md).

For offline or cache-first usage, make sure the local cache already contains activities before setting `useCache=true`.

## Useful Next Reads

- [OAuth and Cache Troubleshooting](../data-sources/strava-oauth-troubleshooting.md)
- [Cache Layout](../architecture/cache-layout.md)
- [OSRM Setup](../routing/osrm-setup.md)
