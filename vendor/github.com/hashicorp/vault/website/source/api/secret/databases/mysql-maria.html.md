---
layout: "api"
page_title: "MySQL/MariaDB Database Plugin - HTTP API"
sidebar_current: "docs-http-secret-databases-mysql-maria"
description: |-
  The MySQL/MariaDB plugin for Vault's Database backend generates database credentials to access MySQL and MariaDB servers.
---

# MySQL/MariaDB Database Plugin HTTP API

The MySQL Database Plugin is one of the supported plugins for the Database
backend. This plugin generates database credentials dynamically based on
configured roles for the MySQL database.

## Configure Connection

In addition to the parameters defined by the [Database
Backend](/api/secret/databases/index.html#configure-connection), this plugin
has a number of parameters to further configure a connection.

| Method   | Path                         | Produces               |
| :------- | :--------------------------- | :--------------------- |
| `POST`   | `/database/config/:name`     | `204 (empty body)` |

### Parameters
- `connection_url` `(string: <required>)` - Specifies the MySQL DSN.

- `max_open_connections` `(int: 2)` - Specifies the maximum number of open
  connections to the database.

- `max_idle_connections` `(int: 0)` - Specifies the maximum number of idle
  connections to the database. A zero uses the value of `max_open_connections`
  and a negative value disables idle connections. If larger than
  `max_open_connections` it will be reduced to be equal.

- `max_connection_lifetime` `(string: "0s")` - Specifies the maximum amount of
  time a connection may be reused. If <= 0s connections are reused forever.

### Sample Payload

```json
{
  "plugin_name": "mysql-database-plugin",
  "allowed_roles": "readonly",
  "connection_url": "root:mysql@tcp(127.0.0.1:3306)/"
  "max_open_connections": 5,
  "max_connection_lifetime": "5s",
}
```

### Sample Request

```
$ curl \
    --header "X-Vault-Token: ..." \
    --request POST \
    --data @payload.json \
    https://vault.rocks/v1/database/config/mysql
```

