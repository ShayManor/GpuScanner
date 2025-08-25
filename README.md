# [GpuScanner](https://gpufindr.com)

A comprehensive GPU monitoring and blog automation system that tracks GPU availability and pricing across multiple cloud providers while generating automated blog content.

https://gpufindr.com

## Overview

GpuScanner is a multi-component Go application that:
- Scans and monitors GPU resources across cloud providers (AWS Lambda, RunPod, TensorDock, Vast.ai)
- Provides a web API and frontend for GPU search and monitoring
- Automatically generates and publishes blog content about GPU market trends
- Integrates with MCP (Model Context Protocol) for AI-powered interactions

## Architecture

The project is organized into several key components:

### Core Applications

- **`cmd/api/`** - Web API server with frontend interface
- **`cmd/blog/`** - Blog content generation and management system
- **`cmd/scan/`** - GPU scanning service across multiple providers

### Key Features

- **Multi-Provider GPU Scanning**: Monitors GPU availability across Lambda, RunPod, TensorDock, and Vast.ai
- **REST API**: Provides endpoints for GPU data retrieval and search
- **Web Frontend**: HTML interface for searching and viewing GPU information
- **Automated Blogging**: Daily blog post generation using AI
- **MCP Integration**: Model Context Protocol support for AI interactions

## Components

### GPU Scanning (`cmd/scan/`)
- `lambdaGetter.go` - AWS Lambda GPU monitoring
- `runpodGetter.go` - RunPod GPU tracking
- `tensordockGetter.go` - TensorDock integration
- `vastGetter.go` - Vast.ai monitoring
- `scan.go` - Main scanning orchestrator

### API Server (`cmd/api/`)
- `main.go` - HTTP server and routing
- `gpu.go` - GPU-related API endpoints
- `blog.go` - Blog management API
- `mcp.go` - MCP protocol implementation
- `static.go` - Static file serving
- `frontend/` - HTML templates and UI

### Blog System (`cmd/blog/`)
- `main.go` - Blog generation entry point
- `write.go` - Content creation and publishing
- `upload.go` - Media and asset management
- `types.go` - Data structures

## Installation

1. Clone the repository:
```bash
git clone https://github.com/ShayManor/GpuScanner.git
cd GpuScanner
```
2. Install dependencies:
```bash
go mod tidy
```
3. Set up environment variables:
```bash
export SUPABASE_URL="example"
export SUPABASE_SERVICE_KEY="example"
export OPENAI_API_KEY="example"
... GPU provider keys
```

## Usage

### Running API
```bash
go run ./cmd/api
```

### Manual GPU scan
```bash
go run ./cmd/scan
```

### Generate Article for Blog
```bash
go run ./cmd/blog
```

## Automation

The project includes GitHub Actions workflow (.github/workflows/blog.yaml) that:
- Runs daily at midnight
- Automatically generates articles
- Publishes to Supabase

## API Documentation

- OpenAPI specification: `cmd/api/openapi.yaml`
- Swagger documentation: `docs/swagger.json`

