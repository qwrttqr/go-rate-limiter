# qwrttqr rate-limiter

This is the implementation of [arxiv2602.11741](https://arxiv.org/pdf/2602.11741) rate limiters approaches and
algorithms.

In-memory cache is implemented as sharded cache on default golang map with mutexes.

## What will be done

1. Admin part with limiting statistic in PostgreSQL
2. Ability to use this rate-limiter as middleware in your service
3. Support of gRPC

## What i am still thinking to add

1. Support of etcd and other distributed data stores

## How to use

### Configuration

#### As standalone service

The service contains one small config file (create your own `config.yaml`):

```yaml
store: "in_memory"
use_algo: "rolling_window"
algo_settings:
  window_size: 100 # seconds
  max_requests: 10
backends:
  in_memory:
    default_ttl: 10 # seconds
    eviction_time: 15 # seconds
    shards: 256
  redis:
    addr: "redis:6379"
    password: "pass"
    db: 0
    default_ttl: 1000
```

- The `store` key is responsible for storage type will be used. Possible options: `in_memory`, `redis`, `etcd`.
- `use_algo` key is responsible for used algo. Possible options `rolling_window`, `token_bucket`, `fixed_window`.
- `cache_settings` allows you to control cache expiration time and (in case of in_memory cache) cache keys eviction
  intervals.
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

Then use `http<s>://<your_host:port>/limit` - for limiting by HTTP.

## Build

If you want to use rate-limiter as standalone service build it as container:
`docker build -t <image-name> .`.

And then run `docker run --rm -p 8080:8080 <image-name>`.

## Benchmarks

You can run bench on your own machine with `make bench COUNT=<N> cPUS=<M>`

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

