# qwrttqr rate-limiter

This is the implementation of [arxiv2602.11741](https://arxiv.org/pdf/2602.11741) rate limiters approaches and
algorithms.

## What already done

1. In memory auto-evicting cache
2. All rate-limiting algos present in the paper in in_memory style

## What will be done

1. Distributed state control via Redis, etcd.
2. Admin part with limiting statistic in PostgreSQL
3. Ability to use this rate-limiter as middleware in your service and as standalone service
4. Support of gRPC

## How to use

### Configuration

The service contains one small config file:

```yaml
configuration:
  store: "in_memory"
  use_algo: "rolling_window"
  cache_settings:
    expiration_time: 10
    eviction_time: 15
  algo_settings:
    window_size: 100 # seconds
    max_requests: 10
```

- The `store` key is responsible for storage type will be used. Possible options: `in_memoty`, `redis`, `etcd`.
- `use_algo` key is responsible for used algo. Possible options `rolling_window`, `token_bucket`, `fixed_window`.
- `cache_settings` allows you to control cache expiration time and (in case of in_memory cache) cache keys eviction
  intervals.
- `algo_settings` key is responsible for configuration for algo:
  - use `window_size` and `max_requests` for `rolling_window` and `fixed_window` algorithms.
  - use `capacity` and `rate` for `token_bucket` algo.