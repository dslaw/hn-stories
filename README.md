# hn-stories

Popularity tracking of Hacker News stories.

Ingest Hacker News stories and their top-level comments at multiple time points,
so that popularity over time can be tracked.


## Getting Started

The project can be run locally using [`k3d`](https://k3d.io/v5.6.3). First,
create and start a `kubernetes` cluster and apply configmaps:
```bash
$ k3d cluster create hn-stories
$ k3d cluster start hn-stories
$ kubectl apply -f manifests/config.yaml -f manifests/database-init-config.yaml
```

Then, start the datastores:
```bash
$ kubectl apply -f manifests/broker.yaml
$ kubectl apply -f manifests/database.yaml
```

Build the project's `docker` image and import it into the `kubernetes` cluster:
```bash
$ docker build . --tag=hn-stories-worker:dev
$ k3d image import hn-stories-worker:dev --cluster hn-stories
```

Finally, run the ingestion workers:
```bash
$ kubectl apply -f manifests/workers.yaml
```


## Development

Run formatting:
```bash
$ gofmt -w -s .
```

Run vetting:
```bash
$ go vet ./...
```

Run tests:
```bash
$ go test ./...
```
