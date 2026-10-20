# qwrttqr rate-limiter

This is the implementation of [arxiv2602.11741](https://arxiv.org/pdf/2602.11741) rate limiters approaches and
algorithms.

In-memory cache is implemented as sharded cache on default golang map with mutexes.

## What i am still thinking to add

1. Support of etcd and other distributed data stores

## Usage modes

The limiter can run in two ways:

| Mode                        | When to use                                                         | Who decides the key           |
|-----------------------------|---------------------------------------------------------------------|-------------------------------|
| **Standalone service**      | Several services share one limiter, or you want it outside your app | The caller sends the key      |
| **Middleware (Go library)** | One Go app, no extra network hop                                    | Your code, via a key function |

## Standalone service

For a standalone service there is available building your own docker compose. Before building please configure service.

> The standalone service trusts the caller to send the key and the token cost.
> It has no authentication, so run it inside a private network and do not expose it to the internet.

### Configuration

The service is configured only through `config.yaml`. Create your own:

```yaml
store: "in_memory"
use_algo: "token_bucket"
metrics:
  use_prometheus: true
  port: 9091
algo_settings:
  capacity: 10
  rate: 0.5            # tokens per second
backends:
  in_memory:
    default_ttl: 10
    eviction_time: 15
    shards: 256
  redis:
    addr: "redis:6379"
    password: "super-secret"
    db: 0
    default_ttl: 1000
```

List of config:

- The `store` key is responsible for storage type will be used. Possible options: `in_memory`, `redis`.
- `use_algo` key is responsible for used algo. Possible options `rolling_window`, `token_bucket`, `fixed_window`.
- `algo_settings` key is responsible for configuration for algo:
    - use `window_size` and `max_requests` for `rolling_window` and `fixed_window` algorithms.
    - use `capacity` and `rate`(per second) for `token_bucket` algo.
- `backend` is responsible for backends configuration:
    - `in_memory` is configuration for in-memory cache:
        - `default_ttl` is default TTL for a key, when cache is touched the TTL is updated.
        - `eviction_time` is time interval in what cache eviction is fired.
        - `shards` is shards count for cache, **KEEP IT AS POWER OF 2**.
    - `redis` is configuration for Redis storage:
        - `addr` is Redis address.
        - `password` is Redis password.
        - `db` DB index inside Redis.
        - `default_ttl`: is a default TTL for Redis cache items.
- `metrics` key is responsible for metrics of standalone service. Prometheus is used for metrics, and you cannot change
  it to another observability framework right now.
    - `use_prometheus` is disabling and enabling usage of Prometheus metrics. it's true or false.
    - `port` is responsible from port that is gonna be used for observability handler.

### API

After build the service is available on ports:

| Port | Protocol |
|------|----------|
| 8080 | HTTP     |
| 8081 | gRPC     |

#### HTTP

`POST /limit`

| Header              | Required | Description                                                    |
|---------------------|----------|----------------------------------------------------------------|
| `X-Client-Key`      | yes      | The identity to limit (for example `user:42` or `ip:1.2.3.4`). |
| `X-Required-Tokens` | no       | Request cost. Default `1`. Must be greater than 0.             |

| Status | Meaning                                                             |
|--------|---------------------------------------------------------------------|
| `200`  | Allowed                                                             |
| `429`  | Limit exceeded. The `Retry-After` header holds the seconds to wait. |
| `400`  | Invalid key or token header                                         |
| `500`  | Limiter backend failure                                             |

```bash
curl -i -X POST http://localhost:8080/limit -H "X-Client-Key: user:42"
```

`GET /health` returns `200 ok`.

#### gRPC

The contract is in [`proto/ratelimiter/v1/ratelimiter.proto`](proto/ratelimiter/v1/ratelimiter.proto).
Generated Go code is committed in `gen/`.

```proto
service RateLimiter {
  rpc Allow(AllowRequest) returns (AllowResponse);
}
```

| Field                       | Description                                                |
|-----------------------------|------------------------------------------------------------|
| `AllowRequest.key`          | Identity to limit.                                         |
| `AllowRequest.tokens`       | Request cost. `0` (unset) means 1. Must be positive.       |
| `AllowResponse.allowed`     | Whether the request may proceed.                           |
| `AllowResponse.retry_after` | How long to wait. Only meaningful when `allowed` is false. |

A denial is a normal response with `allowed = false`, not an error. Errors mean:

| gRPC code          | Meaning                    |
|--------------------|----------------------------|
| `INVALID_ARGUMENT` | Empty  key, invalid tokens |
| `UNAVAILABLE`      | Limiter backend failure    |

Example with grpcurl:

```bash
grpcurl -plaintext -import-path proto -proto ratelimiter/v1/ratelimiter.proto \
  -d '{"key":"user:42"}' localhost:8081 ratelimiter.v1.RateLimiter/Allow
```

Go client:

```go
conn, _ := grpc.NewClient("localhost:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
defer conn.Close()

c := ratelimiterv1.NewRateLimiterClient(conn)
resp, err := c.Allow(ctx, &ratelimiterv1.AllowRequest{Key: "user:42"})
if err == nil && !resp.GetAllowed() {
time.Sleep(resp.GetRetryAfter().AsDuration())
}
```

For other languages, generate a client from the `.proto` file.

## Middleware (Go library)

### Create a limiter

A limiter is built from a backend (where counters live) and an algorithm.

```go
backend := middleware.NewInMemoryBackend(config.InMemoryBackendSettings{
DefaultTtl: 10, EvictionTime: 15, ShardsCount: 256, // shards: power of 2
})
// or: middleware.NewRedisBackend(redisClient, 10)

capacity, rate := 10.0, 0.5
limiter, err := backend.TokenBucket(config.AlgoSettings{Capacity: &capacity, Rate: &rate})
if err != nil {
log.Fatal(err)
}
```

`backend.FixedWindow(...)` and `backend.RollingWindow(...)` work the same way (`WindowSize` and `MaxRequests`).

### HTTP

```go
mw := middleware.NewHTTPMiddleware(
limiter,
middleware.HTTPKeyRemoteIP, // who is limited
nil,                        // cost per request, nil = 1
false,                      // failOpen
)
http.ListenAndServe(":8080", mw(yourHandler))
```

### gRPC

```go
srv := grpc.NewServer(
grpc.ChainUnaryInterceptor(
middleware.NewGRPCUnaryInterceptor(limiter, middleware.GRPCKeyRemoteIP, nil, false)),
grpc.ChainStreamInterceptor(
middleware.NewGRPCStreamInterceptor(limiter, middleware.GRPCKeyRemoteIP, nil, false)),
)
```

A stream is counted once, when it is opened, not per message.
Clients read the trailer with the `grpc.Trailer(&md)` call option.

### Keys

The key decides who shares a bucket. Built-in functions:

| Function                                                   | Key                                   |
|------------------------------------------------------------|---------------------------------------|
| `HTTPKeyRemoteIP`, `GRPCKeyRemoteIP`                       | Client IP (IPv6 is grouped by /64)    |
| `HTTPKeyAuthorizationHeader`, `GRPCKeyAuthorizationHeader` | SHA-256 of the `Authorization` header |

Write your own for anything else (user ID, tenant, route):

```go
func keyByUser(r *http.Request) (string, error) {
return "user:" + userIDFromContext(r.Context()), nil
}
```

> **IP keys** use the address of the direct connection. Behind a load balancer
> or reverse proxy every client looks the same, so write a key function that
> reads the forwarded address your proxy sets, and only trust that header when
> it comes from your own proxy.
>
> **Authorization keys** do not validate the token. Put the middleware after validation or write custom key derivation
> function with validation.

### Cost

Pass a function to make some requests more expensive. It must return a value
greater than 0.

```go
cost := func (r *http.Request) (float64, error) {
if r.Method == http.MethodPost {
return 5, nil
}
return 1, nil
}
```

### Fail open or closed

`failOpen` decides what happens when the limiter cannot answer (backend down,
key or cost function returned an error, invalid cost).

- `false`: reject the request (`503` / `UNAVAILABLE`).
- `true`: let it through and log.

## Metrics

As it was said before metrics is supported only in standalone service mode. And Prometheus metrics is available through
`/GET metrics` handler.

The metrics key that service give access to:

| Metric                                  | Type      | Explanation                                                     |
|-----------------------------------------|-----------|-----------------------------------------------------------------|
| `ratelimiter_decisions_total`           | Counter   | Total count by each decision: `allowed`, `denied` and `errored` |
| `ratelimiter_decision_duration_seconds` | Histogram | Histogram for each decision type.                               |

## Benchmarks

You can run bench on your own machine with `make bench COUNT=<N> CPUS=<M>`

Benchmarks simulates 3 scenarios:

1. One single hot key.
2. Uniform distribution for count of requests between keys.
3. Zipf distribution - some keys gets many requests and becoming hot keys, other gets small count.

The benches:

```text
go test -run='^$' -bench=. -benchmem -count=1 -cpu=1,4,8,12 ./bench
goos: windows
goarch: amd64
pkg: qwrttqr-rate-limiter/bench
cpu: AMD Ryzen 5 7500F 6-Core Processor             
BenchmarkFixedWindow/single_key/allow                   56308683                21.09 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/single_key/allow-4                 32228694                36.01 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/single_key/allow-8                 21281122                49.20 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/single_key/allow-12                21743620                50.77 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/single_key/deny                    57306315                21.69 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/single_key/deny-4                  35630404                32.10 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/single_key/deny-8                  21430484                54.28 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/single_key/deny-12                 19749672                60.63 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/allow                 17826820               100.1 ns/op             0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/allow-4               38755827                30.41 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/allow-8               70427080                17.10 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/allow-12              93669501                12.46 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/deny                  19345477                62.61 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/deny-4                36055633                31.32 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/deny-8                64787469                17.84 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/uniform_100k/deny-12               89258484                12.93 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/allow                    44851597                33.13 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/allow-4                  25398061                45.11 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/allow-8                  21211081                53.46 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/allow-12                 22018509                53.20 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/deny                     45966091                33.09 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/deny-4                   26478434                46.47 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/deny-8                   22155386                54.79 ns/op            0 B/op          0 allocs/op
BenchmarkFixedWindow/zipf_100k/deny-12                  21086331                58.82 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/allow                 48130633                24.26 ns/op           48 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/allow-4               27737712                42.83 ns/op           43 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/allow-8               18548516                65.87 ns/op           41 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/allow-12              17378282                70.37 ns/op           44 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/deny                  49189197                22.18 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/deny-4                32335164                37.42 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/deny-8                20683236                57.47 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/single_key/deny-12               19619544                61.06 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/allow               11891265               163.4 ns/op            22 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/allow-4             31139793                54.78 ns/op           22 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/allow-8             59801160                31.52 ns/op           22 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/allow-12            78769100                22.02 ns/op           23 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/deny                10065349               120.7 ns/op             0 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/deny-4              29053438                38.84 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/deny-8              57521126                21.41 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/uniform_100k/deny-12             76720455                14.73 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/allow                  29967036                33.76 ns/op           42 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/allow-4                24476508                50.26 ns/op           42 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/allow-8                21045025                56.36 ns/op           45 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/allow-12               18872197                59.69 ns/op           42 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/deny                   43794181                26.95 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/deny-4                 27759463                44.86 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/deny-8                 22893991                53.38 ns/op            0 B/op          0 allocs/op
BenchmarkRollingWindow/zipf_100k/deny-12                20852665                55.23 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/allow                   50584034                21.54 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/allow-4                 31128162                37.97 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/allow-8                 20739536                56.76 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/allow-12                19343419                59.92 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/deny                    51248104                21.86 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/deny-4                  29546898                40.51 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/deny-8                  19755330                58.78 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/single_key/deny-12                 18852451                61.90 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/allow                 18177328                74.44 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/allow-4               32287052                33.88 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/allow-8               57865346                19.32 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/allow-12              69705435                14.38 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/deny                  15133953                70.50 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/deny-4                37284100                31.37 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/deny-8                67392634                17.43 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/uniform_100k/deny-12               88353531                13.63 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/allow                    42428763                31.25 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/allow-4                  26150000                44.76 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/allow-8                  21181690                53.84 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/allow-12                 21550485                54.99 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/deny                     44520292                30.87 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/deny-4                   26432766                45.39 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/deny-8                   21224098                54.54 ns/op            0 B/op          0 allocs/op
BenchmarkTokenBucket/zipf_100k/deny-12                  21434962                56.57 ns/op            0 B/op          0 allocs/op
```


