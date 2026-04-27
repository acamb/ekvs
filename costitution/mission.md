# EKVS - Easy Key-Value Store

This project aims to provide an easy solution to manage encrypted key-value stores.
The encryption is always done client-side, so the server will never have access to the unencrypted data.

To keep the process simple, users will be authenticated only using a ssh key pair, so there will be no need to manage passwords or other authentication methods.
Each user will have their own key-value store, and each store can have multiple projects containing key-value pairs.

The main goal is to manage secrets in a simple and secure way using 3 main components:
* the server
* the TUI client for managing the key-value stores
* the CLI client for secret provisioning, supporting various methods (export to environment, write to pipe, etc...)

The goal of the CLI client is to provision secrets to other applications supporting various deployment scenarios, such as CI pipelines, containers, standalone applications, etc...