# Gofasta Repo
## Motivation
In the evolving landscape of web development, GraphQL has emerged as a powerful alternative to REST APIs, offering more flexibility and efficiency in querying data. However, many existing Golang packages and templates are primarily focused on REST APIs, with limited support for integrating GraphQL functionalities, especially when using gqlgen.

Recognizing this gap, the gofasta project was created to streamline the development of GraphQL APIs in Golang. Our goal is to provide a robust, full-featured template that simplifies the setup and configuration of a GraphQL server, enabling developers to focus on building and optimizing their applications rather than wrestling with boilerplate code.

This template includes pre-configured packages and best practices tailored specifically for GraphQL development. Once the server is up and running, the API will be accessible at the /graphql endpoint, providing a solid foundation for building scalable and efficient GraphQL services.

By leveraging this template, developers can accelerate their project setup, maintain high code quality, and integrate GraphQL seamlessly into their Golang applications.

## Folder structure
```go
├── app
│   ├── cmd
│   │   ├── db-init.go
│   │   ├── main.go
│   │   └── server.go
│   ├── dtos
│   │   ├── common.dtos.go
│   │   ├── generated-types.dtos.go
│   │   └── user.dtos.go
│   ├── graphql
│   ├── resolvers
│   │   ├── resolver.go
│   │   ├── server.health.resolvers.go
│   │   └── user.resolvers.go
│   ├── schema
│   │   ├── common.gql
│   │   ├── server.health.gql
│   │   └── user.gql
│   ├── models
│   │   ├── base.model.go
│   │   └── user.model.go
│   ├── rest
│   │   ├── controllers
│   │   │   ├── controller.go
│   │   │   └── user.controller.go
│   │   ├── routes
│   │   │   ├── index.routes.go
│   │   │   └── user.routes.go
│   ├── services
│   │   └── user.service.go
│   ├── utils
│   │   ├── build-search-query.go
│   │   ├── convert-struct-to-map.go
│   │   ├── paginator.go
│   │   ├── random-password.go
│   │   ├── string.go
│   │   └── validator.go
│   ├── validators
│   │   ├── common.validators.go
│   │   ├── custom-messages.validators.go
│   │   ├── register.validators.go
│   │   ├── user.validators.go
│   │   ├── validate-input.go
│   │   ├── validator.utils.go
│   │   └── generated.go
├── configs
│   └── database.go
├── db
│   ├── migrations
│   │   ├── 000001_create_citext_extension.down.sql
│   │   ├── 000001_create_citext_extension.up.sql
│   │   ├── 000002_create_function_to_update_updated_at_column.down.sql
│   │   ├── 000002_create_function_to_update_updated_at_column.up.sql
│   │   ├── 000003_create_function_to_avoid_duplicate_records.down.sql
│   │   ├── 000003_create_function_to_avoid_duplicate_records.up.sql
│   │   ├── 000004_increment_record_version.down.sql
│   │   ├── 000004_increment_record_version.up.sql
│   │   ├── 000005_create_users.down.sql
│   │   └── 000005_create_users.up.sql
├── deployments
│   ├── dev
│   │   ├── app
│   │   │   ├── dev_entrypoint.sh
│   │   │   ├── dockerfile
│   │   │   └── wait-for-it.sh
│   │   ├── db
│   │   │   └── dockerfile
│   ├── production
│   │   ├── app
│   │   │   ├── compose-production.yml
│   │   │   ├── deploy-production.sh
│   │   │   ├── dockerfile
│   │   │   ├── gofasta-production-process.sh
│   │   │   ├── production_entrypoint.sh
│   │   │   └── wait-for-it.sh
│   │   ├── db
│   │   │   ├── compose-production.yml
│   │   │   ├── dockerfile
│   │   │   └── gofasta-production-postgresql-db-process.sh
│   │   ├── server
│   │   │   ├── proxy-reverse-config
│   │   │   │   ├── gofasta-qa.ironji.com.conf
│   │   │   │   ├── .gitkeep
│   │   │   │   └── .gitkeep
│   ├── qa
│   │   ├── app
│   │   │   ├── compose-qa.yml
│   │   │   ├── deploy-qa.sh
│   │   │   ├── dockerfile
│   │   │   ├── gofasta-qa-process.sh
│   │   │   ├── qa_entrypoint.sh
│   │   │   └── wait-for-it.sh
│   │   ├── db
│   │   │   ├── compose-qa.yml
│   │   │   ├── dockerfile
│   │   │   └── gofasta-qa-postgresql-db-process.sh
│   │   ├── server
│   │   │   ├── proxy-reverse-config
│   │   │   │   ├── gofasta-qa.ironji.com.conf
│   │   │   │   ├── .gitkeep
│   │   │   │   └── .gitkeep
├── scripts
│   ├── connect-to-api.sh
│   ├── connect-to-db.sh
│   ├── generate-migration.sh
│   ├── migrate-down.sh
│   └── migrate-up.sh
├── tmp
│   ├── build-errors.log
├── .air.toml
├── .env
├── .env.sample
├── .gitignore
├── compose-dev.yml
├── go.mod
├── go.sum
├── gqlgen.yml
├── README.md
├── start.sh
└── tools.go
```

### Directory and File Descriptions
`app/`: Contains the core application code.

- `cmd/`: Contains the entry point files for the application.

    - `db-init.go`: Initialization script for the database.
    - `main.go`: Main entry point of the application.
    - `server.go`: Server configuration and startup code.
    - `graphql/`: GraphQL schema and data transfer objects (DTOs).

        - `dtos/`:
            - `common.dtos.go`: Common data transfer objects.
            - `generated-types.dtos.go`: Auto-generated types for GraphQL.
            - `user.dtos.go`: User-related data transfer objects.
        - `common.gql`: Common GraphQL schema.
        - `server.health.gql`: Server health check schema.
        - `user.gql`: User-related GraphQL schema.
    - `models/`: Database models.

        - `base.model.go`: Base model definitions.
        - `user.model.go`: User model definitions.
    - `resolvers/`: GraphQL resolvers.

        - `resolver.go`: Main resolver file.
        - `server.health.resolvers.go`: Resolvers for server health checks.
        - `user.resolvers.go`: User-related resolvers.
    - `services/`: Service layer for business logic.

        - `user.service.go`: User-related business logic.
    - `utils/`: Utility functions and helpers.

        - `build-search-query.go`: Functions for building search queries.
        - `convert-struct-to-map.go`: Functions for converting structs to maps.
        - `paginator.go`: Pagination utility.
        - `random-password.go`: Random password generator.
        - `string.go`: String manipulation utilities.
        - `validator.go`: Validation utilities.
    - `generated.go`: Auto-generated code, this file is auto-generated by `gqlgen` a package we use for graphql by running this command:
    ```sh
    go run github.com/99designs/gqlgen generate
    ```
    You will have to run that command always whenever you are done with modifying any file with `.gql` extension from `graphql/` folder

- `configs/`
Configuration files.

    - `database.g`o: Database configuration.
- `db/`
Database-related files.

    - `migrations/`: Database migration files.
        -     `000001_create_function_to_update_updated_at_column.down.sql`: Down migration for updating updated_at column function.
        -   `000001_create_function_to_update_updated_at_column.up.sql`: Up migration for updating updated_at column function.
        - `000002_create_users.down.sql`: Down migration for creating users table.
        - `000002_create_users.up.sql`: Up migration for creating users table.
- `deployments/`
Deployment-related files.

    - `dev/`: Development environment configurations.

        - `app/`:
            - `dev_entrypoint.sh`: Development entry point script.
            - `dockerfile`: Dockerfile for the app.
            - `wait-for-it.sh`: Script to wait for dependencies to be ready.
        - `db/`:
            - `dockerfile`: Dockerfile for the database.
    - `prod/`: Production environment configurations.

    - `qa/`: QA environment configurations.

- `scripts/`
Helper scripts for common tasks.

    - `connect-to-api.sh`: Script to connect to the API container.
    - `connect-to-db.sh`: Script to connect to the database container.
    - `generate-migration.sh`: Script to generate database migrations.
    - `migrate-down.sh`: Script to migrate the database down.
    - `migrate-up.sh`: Script to migrate the database up.
- `tmp/`
Temporary files.

#### Root Files
- `.air.toml`: Air configuration for live reloading.
- `.env`: Environment variables.
- `.gitignore`: Git ignore file.
- `compose-dev.yml`: Docker Compose file for development.
- `go.mod`: Go modules file.
- `go.sum`: Go modules checksum file.
- `gqlgen.yml`: gqlgen configuration file.
- `README.md`: Project readme file.
- `start.sh`: Script to start the application.
- `tools.go`: Tools for the project.

