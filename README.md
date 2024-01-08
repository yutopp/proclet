# Proclet

[Proclet](https://code.yutopp.net/) is an online compiler service. Currently heavely under development.

## Architecture

Frontend is implemented by `TypeScript+React`, and backend is implemented by `Go`.

Communications are done by `Connect`. Frontend and backend can be deployed by `Docker`.

Backend uses Docker daemon on the host machine to compile and execute source codes.

Currently, this system is designed to be hosted on a single machine.
