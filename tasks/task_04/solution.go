package main

import "math"

type Stats struct {
	Count         int
	Sum, Min, Max int64
}

func Calc(nums []int64) Stats {
	var result = Stats{}
	if len(nums) < 2 {
		return result
	}

	result.Min = math.MaxInt64
	result.Max = math.MinInt64
	for i := range nums {
		if i > 0 {
			result.Sum += nums[i] - nums[i-1]
			result.Count += 1
			result.Min = min(result.Min, nums[i]-nums[i-1])
			result.Max = max(result.Max, nums[i]-nums[i-1])
		}
	}
	return result
}
