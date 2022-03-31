### Simple Gitlab "crawler"

- `service.go` - crawler with limited number of network-bounded and cpu-bounded goroutines and graceful termination.
- `service_test.go` - "intergration test" for crawler with mocked GraphQL API client.
- `gitlab_graphql.go` - implementation of simple Gitlab GRAPHQL API client.
- `main.go` - crawler demonstration.