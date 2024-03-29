# Rate Limiter

## Overview

This project implements a web service that acts as a third-party rate limiter service. The rate limiter allows clients to report URL visits and checks whether the number of visits for each URL has exceeded the specified threshold within a given time period (ttl). If the threshold is reached, the service blocks further requests for that URL.

## Instructions

1. To run the rate limiter service, execute the following command:

   ```bash
   go run main.go -threshold 10 -ttl 1000 -capacity=1000 -workers=20
   ```

   The command-line arguments `-threshold`, `-ttl`, `-capacity`, and `-workers` allow you to customize the behavior of the rate limiter.

2. The rate limiter service exposes an HTTP endpoint at `/report`, which accepts URL visits in JSON format. To report a URL visit, make a POST request to the `/report` endpoint with the following JSON body:

   ```json
   {
     "url": "http://www.sample.com"
   }
   ```

3. The service will respond with a JSON object containing `"block": true` if the number of times the URL was reported has reached the threshold within the specified `ttl` period. Otherwise, the response will be `"block": false`.

4. The rate limiter uses a buffered channel to manage the queue of requests and processes them concurrently using a specified number of worker goroutines.

5. The service utilizes a simple hash conversion to store and track URL visits efficiently.

6. URLs are hashed to reduce memory usage, and any invalid URLs will be rejected.

## Alternatives with Go Modules

- Instead of using the standard HTTP package, consider using `fastHTTP` for improved performance.
- Replace the standard `log` package with `zap logger` for more robust and configurable logging capabilities.
- Instead of using command-line flags for arguments, consider using `viper` to read configuration from environment variables.

## Profiling

To enable profiling, uncomment the following code in the `main` function:

```go
/*go func() {
  log.Println(http.ListenAndServe("localhost:6060", nil))
}()*/
```

You can then use various commands to profile the service:

- `go tool pprof -png http://localhost:6060/debug/pprof/allocs > allocs.png`: Profiling heap allocations.
- `go tool pprof -png http://localhost:6060/debug/pprof/mutex > mutex.png`: Profiling mutex contention.
- `go tool pprof -png http://localhost:6060/debug/pprof/heap > heap.png`: Profiling heap allocations.
