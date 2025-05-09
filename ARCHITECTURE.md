# S3Proxy Architecture

## Overview

S3Proxy is a lightweight proxy for S3-compatible object storage that provides simplified access to objects and file listings. This document outlines the architecture of the application, which follows Clean Architecture principles with Domain-Driven Design (DDD).

## Architectural Layers

The application is organized into four main layers:

1. **Domain Layer**: Contains the core business logic and entities
2. **Application Layer**: Implements use cases using domain entities
3. **Infrastructure Layer**: Provides implementations for external services
4. **Interface Layer**: Handles external interactions like HTTP requests

### Layer Dependencies

Each layer depends only on layers closer to the core:

```
Interface → Application → Domain ← Infrastructure
```

## Domain Layer

The domain layer represents the core business logic of S3Proxy:

- **Entities**: Core business objects 
  - `S3Object`: Represents a file in S3
  - `ObjectCollection`: Collection of S3 objects with filtering and search capabilities

- **Repository Interfaces**: Define interactions with external systems
  - `S3Repository`: Interface for S3 storage operations

## Application Layer

The application layer contains use cases and orchestrates the domain entities:

- **DTOs**: Data Transfer Objects for interactions between layers
  - `ObjectDTO`: Simplified object representation for clients
  - `ListResponseDTO`: Response for listing operations
  - `DirectoryDTO`: Directory with files for UI

- **Use Cases**: Application-specific business rules
  - `ListFilesUseCase`: Handles listing objects in S3
  - `ProxyFileUseCase`: Handles redirecting to objects in S3

## Infrastructure Layer

The infrastructure layer provides implementations for external services:

- **Repositories**: Implementations of domain repository interfaces
  - `S3RepositoryImpl`: Implements `S3Repository` using MinIO SDK

- **Configuration**: Application configuration
  - `Config`: Configuration from environment variables

## Interface Layer

The interface layer handles external interactions:

- **HTTP**: Web server and handlers
  - `Server`: Fiber web server
  - `Handler`: HTTP request handlers

## Flow of Control

1. HTTP requests are received by the interface layer
2. Handlers invoke appropriate use cases in the application layer
3. Use cases orchestrate domain entities
4. Domain entities access external systems via repository interfaces
5. Infrastructure layer provides implementations of repository interfaces
6. Results flow back through the layers to the client

## Benefits of This Architecture

- **Separation of Concerns**: Each layer has a specific responsibility
- **Testability**: Business logic can be tested in isolation
- **Flexibility**: External integrations can be changed without affecting business logic
- **Maintainability**: Clear structure makes the codebase easier to understand and maintain

## Example: File Proxy Use Case

When a user requests a file:

1. HTTP server routes the request to `ProxyFile` handler
2. Handler invokes `ProxyFileUseCase.Execute`
3. Use case validates inputs and calls `S3Repository.FindObject` and `S3Repository.GetPresignedURL`
4. Repository implementation queries S3 service for the object
5. Use case validates the result and returns the presigned URL
6. Handler redirects the client to the presigned URL