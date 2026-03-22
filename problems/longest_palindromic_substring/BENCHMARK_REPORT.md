# Longest Palindromic Substring Benchmark Report

## Summary

The solution was refactored from a stateful approach that accumulated intermediate palindrome descriptors into a direct `expand-around-center` implementation.

This removed all benchmarked heap allocations and improved throughput across every measured scenario.

## Scope

Files changed:

- `problems/longest_palindromic_substring/solution.go`
- `problems/longest_palindromic_substring/solution_test.go`

Benchmark coverage:

- Short canonical inputs
- Medium mixed-content input
- Worst-case repeated-character inputs
- Long input with no long palindrome runs

## Method

Commands used:

```bash
go test ./problems/longest_palindromic_substring
go test ./problems/longest_palindromic_substring -run ^$ -bench BenchmarkLongestPalindrome -benchmem
go test ./problems/longest_palindromic_substring -run ^$ -gcflags=-m=2
```

Comparison numbers below were collected by running a temporary side-by-side benchmark of the legacy and current implementations with `-benchmem` on the same machine.

Environment:

- OS: Linux
- Architecture: amd64
- CPU: AMD Ryzen 7 5800X 8-Core Processor

## Results

| Case | Legacy ns/op | Current ns/op | Speedup | Legacy B/op | Current B/op | Legacy allocs/op | Current allocs/op |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| `canonical_babad` | 103.5 | 15.64 | 6.6x | 192 | 0 | 2 | 0 |
| `canonical_cbbd` | 103.2 | 13.35 | 7.7x | 192 | 0 | 2 | 0 |
| `mixed_medium` | 485.7 | 48.53 | 10.0x | 2112 | 0 | 5 | 0 |
| `repeated_128` | 17803 | 5868 | 3.0x | 19008 | 0 | 8 | 0 |
| `repeated_1024` | 814893 | 292120 | 2.8x | 193088 | 0 | 12 | 0 |
| `no_long_runs_1040` | 17211 | 2802 | 6.1x | 119360 | 0 | 11 | 0 |

## Findings

1. The current implementation eliminates all measured heap allocations in every benchmark scenario.
2. The largest wins come from removing the intermediate `[]Polindrome` growth and avoiding per-entry storage of the input string.
3. CPU time still grows significantly on repeated-character strings because `expand-around-center` remains `O(n^2)` in the worst case.
4. For realistic short and medium inputs, the new implementation is dramatically faster and materially simpler.

## Allocation Analysis

The old implementation allocated due to:

- Growth of the intermediate palindrome slice
- Additional struct values carrying the original string
- Extra work around helper-based boundary checks in the hot path

The current implementation:

- Scans centers directly
- Tracks only best start and end indexes
- Returns a slice of the original string without copying

Escape analysis after the refactor showed no internal heap allocation in `longestPalindrome`; the only retained escape is the returned substring view, which is expected behavior.

## Conclusion

The refactor is a clear performance win:

- Lower latency in all benchmarked cases
- Zero measured allocations
- Smaller and easier-to-maintain implementation

The remaining tradeoff is algorithmic worst-case CPU behavior, which is acceptable for the current LeetCode-style constraint set and substantially better than the previous implementation in practice.
