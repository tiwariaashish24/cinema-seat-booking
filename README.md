### Load Testing

- Tested up to 100,000 goroutines.
- Windows socket buffer exhaustion occurred under
  extreme simultaneous Redis connection activity.
- Reduced concurrent test workload to 1,000 for
  stable and clean race-detector testing.
- Future work: bounded worker-pool testing and
  Redis connection-pool optimization.
