# go-workflow-example

## Getting started

### Building the source code

```bash
$ make build
```

## List of environment variables

Configuration that varies between deployments should be stored in the environment.
This is suggested by the twelve-factor app methodology [https://12factor.net/config](https://12factor.net/config).

| Name | Description |
| --- | --- |
| PORT | Optional. The port to be listen to. If not set, the default port (8080) will be used. |
| CADENCE_HOST | The hostname of the cadence instance. |
| CADENCE_PORT | The port of the cadence instance. |
| CADENCE_DOMAIN | The domain of the cadence instance. |
| CADENCE_TASK_LIST_NAME | The task list name of the cadence instance. |
| CADENCE_WORKFLOW_NAME | The workflow name of the cadence instance. |

## REST resources

### HTTP POST /workflow

**Request**

```json
{
  "name": "string",
  "waitingTime": "number"
}
```

**Response**

```json
{
  "id": "string"
}
```

**Example**

```bash
$ curl -v -d '{"name": "example-name", "waitingTime": 45}' \
    -H "Content-Type: application/json" \
    -X POST http://localhost:8080/workflow
```

### HTTP GET /workflow/status

**Example**

```bash
$ curl -v -X GET http://localhost:8080/workflow/status?id=example-name
```

### HTTP GET /workflow/result

**Example**

```bash
$ curl -v -X GET http://localhost:8080/workflow/result?id=example-name
```

## Testing

Initialize a Cadence instance locally

```bash
$ ./debug
```

```bash
$ make test
```