# TreveccaPedia (trevecca-pedia)

TreveccaPedia is a centralized wiki-based information repository for all things Trevecca Nazarene University.

The goal is to provide a unified and common platform for the publishing, sharing, and accessing of information, to serve an ever-growing need in the Trevecca community.

# Usage/Development

Be sure to install the necessary tools from the [Common Tools](#common-tools) section.

If you want to change or develop certain parts of the project, you'll need tools that are described in the [Developer Tools](#developer-tools) section.

Each service is located in its own directory.

Inside each service directory is a `README.md` for info on deploying and using that service.

- [Web Server](./web/README.md)
- [API Layer](./api-layer/README.md)
- [Authentication Service](./auth/README.md)
- [Wiki](./wiki/README.md)
- [Wiki Database](./wiki-db/README.md)
- [Wiki Filesystem](./wiki-fs/README.md)
- [Search](./search/README.md)

We also have a more streamlined Docker Compose setup for local development.  There are a lot of services involved, so we highly recommend it.
You can read more about it in our [Development Docs](./docs/DEVELOPMENT.md).

## Common Tools

You will have to install these in order to run the services.

For Windows, I recommend using [WSL](https://learn.microsoft.com/en-us/windows/wsl/install) and installing the Linux versions.
Otherwise, Linux is the preferred environment, and MacOS should work as well.

---
### Go 1.25+

[https://go.dev/](https://go.dev)

Install using the instructions here: [https://www.docker.com/](https://go.dev/doc/install).

---
### air-verse/air

[https://github.com/air-verse/air](https://github.com/air-verse/air)

Install using Go:
```
go install github.com/air-verse/air@latest
```

---
### Docker

[https://www.docker.com/](https://www.docker.com)

Install the Docker Engine from [https://docs.docker.com/engine/install/](https://docs.docker.com/engine/install/).

For MacOS, you might have to install Docker Desktop to get this working. The interaction with Docker is typically through the command line, though.

---
### PostgreSQL (v18)

[https://www.postgresql.org/](https://www.postgresql.org/)

Install PostgreSQL from [https://www.postgresql.org/download/](https://www.postgresql.org/download/).

This project mainly utilizes the command-line utility `psql` for testing databases, since the actual database is deployed using Docker.

---
### Node/Bun

[https://nodejs.org/en](https://nodejs.org/en)

Install Node from [https://docs.npmjs.com/downloading-and-installing-node-js-and-npm](https://docs.npmjs.com/downloading-and-installing-node-js-and-npm).

OR

[https://bun.com/](https://bun.com/)

Install Bun from [https://bun.com/](https://bun.com/)

---


