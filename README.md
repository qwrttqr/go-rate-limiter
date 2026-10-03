# qwrttqr rate-limiter

This is the implementation of [arxiv2602.11741](https://arxiv.org/pdf/2602.11741) rate limiters approaches and
algorithms.

## What already done

1. In memory auto-evicting cache
2. All rate-limiting algos present in the paper are recreated with in_memory style

## What will be done

1. Distributed state control via etcd.
2. Admin part with limiting statistic in PostgreSQL
3. Ability to use this rate-limiter as middleware in your service
4. Support of gRPC

## How to use

### Configuration

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
  redis:
    addr: "redis:6379"
    password: "pass"
    db: 0
    default_ttl: 1000
```

- The `store` key is responsible for storage type will be used. Possible options: `in_memoty`, `redis`, `etcd`.
- `use_algo` key is responsible for used algo. Possible options `rolling_window`, `token_bucket`, `fixed_window`.
- `cache_settings` allows you to control cache expiration time and (in case of in_memory cache) cache keys eviction
  intervals.
- `algo_settings` key is responsible for configuration for algo:
    - use `window_size` and `max_requests` for `rolling_window` and `fixed_window` algorithms.
    - use `capacity` and `rate`(per second) for `token_bucket` algo.

Then use `http<s>://<your_host:port>/limit` - for limiting by HTTP.

## Build

If you want to use rate-limiter as standalone service build it as container:
`docker build -t <image-name> .`.

And then run `docker run --rm -p 8080:8080 <image-name>`.

## Benchmarks

## HTTP

### Setup

- **Machine:** a single desktop, Ryzen 5 7500F (6 cores, 12 threads), Docker on Windows. The load generator and the
  server ran on the same machine, so they compete for CPU.
- **Server container:** 3 CPUs. **Vegeta container:** 6 CPUs.
- **Load:** 30,000 req/s fixed rate, 30 s per run.
- **Modes:** `baseline` (`DO_ALGO=0`) goes through the full HTTP stack but skips the limiter, `limiter` (`DO_ALGO=1`)
  runs the algorithm.

Scenarios:

- `uniform100k`: 100,000 keys, uniform distribution. Every request is allowed.
- `zipf`: 100,000 keys, Zipf distribution (a few hot keys, long tail of cold ones). Mixed allowed and rejected.
- `single`: one key. After 500 requests everything is rejected.

```shell
vegeta attack -targets=/targets/${SCENARIO}.txt -rate=${RATE} -duration=${DURATION:-30s} -workers=500 -max-workers=2000 -connections=2000
```

### Rolling window

```yaml
store: "in_memory"
use_algo: "rolling_window"
algo_settings:
  window_size: 60 # seconds
  max_requests: 500
backends:
  in_memory:
    default_ttl: 30 # seconds
    eviction_time: 120 # seconds
  redis:
    addr: "redis:6379"
    password: "PASS"
    db: 0
    default_ttl: 1000
```

#### Results

##### p50 latency (µs)

| mode                   | uniform100k | zipf  | single |
|------------------------|-------------|-------|--------|
| baseline (`DO_ALGO=0`) | 157.1       | 156.9 | 154.5  |
| limiter (`DO_ALGO=1`)  | 160.8       | 164.3 | 167.5  |

##### p95 latency (µs)

| mode                   | uniform100k | zipf  | single |
|------------------------|-------------|-------|--------|
| baseline (`DO_ALGO=0`) | 320.9       | 323.7 | 288.7  |
| limiter (`DO_ALGO=1`)  | 373.4       | 456.8 | 377.6  |

##### p99 latency (ms)

| mode                   | uniform100k | zipf | single |
|------------------------|-------------|------|--------|
| baseline (`DO_ALGO=0`) | 15.9        | 19.5 | 2.1    |
| limiter (`DO_ALGO=1`)  | 13.2        | 32.3 | 4.3    |

### Fixed window

```yaml
store: "in_memory"
use_algo: "fixed_window"
algo_settings:
  window_size: 60 # seconds
  max_requests: 500
backends:
  in_memory:
    default_ttl: 30 # seconds
    eviction_time: 120 # seconds
  redis:
    addr: "redis:6379"
    password: "PASS"
    db: 0
    default_ttl: 1000
```

#### Results

##### p50 latency (µs)

| mode                   | uniform100k | zipf  | single |
|------------------------|-------------|-------|--------|
| baseline (`DO_ALGO=0`) | 157.4       | 154.9 | 154.4  |
| limiter (`DO_ALGO=1`)  | 156.1       | 162.8 | 164.4  |

##### p95 latency (µs)

| mode                   | uniform100k | zipf  | single |
|------------------------|-------------|-------|--------|
| baseline (`DO_ALGO=0`) | 332.5       | 295.3 | 290.2  |
| limiter (`DO_ALGO=1`)  | 305.6       | 382.5 | 391.8  |

##### p99 latency (ms)

| mode                   | uniform100k | zipf | single |
|------------------------|-------------|------|--------|
| baseline (`DO_ALGO=0`) | 15.4        | 2.5  | 2      |
| limiter (`DO_ALGO=1`)  | 1.8         | 7.1  | 16.1   |

### Token bucket

```yaml
store: "in_memory"
use_algo: "token_bucket"
algo_settings:
  capacity: 120 # tokens
  rate: 2 # tokens per second
backends:
  in_memory:
    default_ttl: 30 # seconds
    eviction_time: 120 # seconds
  redis:
    addr: "redis:6379"
    password: "PASS"
    db: 0
    default_ttl: 1000
```

#### Results

##### p50 latency (µs)

| mode                   | uniform100k | zipf  | single |
|------------------------|-------------|-------|--------|
| baseline (`DO_ALGO=0`) | 158.3       | 167.4 | 156.9  |
| limiter (`DO_ALGO=1`)  | 158.8       | 161.4 | 163.5  |

##### p95 latency (µs)

| mode                   | uniform100k | zipf  | single |
|------------------------|-------------|-------|--------|
| baseline (`DO_ALGO=0`) | 320.2       | 917.8 | 319.0  |
| limiter (`DO_ALGO=1`)  | 330.3       | 334.3 | 345.6  |

##### p99 latency (ms)

| mode                   | uniform100k | zipf | single |
|------------------------|-------------|------|--------|
| baseline (`DO_ALGO=0`) | 2.4         | 12.9 | 2.3    |
| limiter (`DO_ALGO=1`)  | 4.5         | 2.8  | 2.5    |
